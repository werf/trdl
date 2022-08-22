package tuf

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/theupdateframework/go-tuf"
	tufClient "github.com/theupdateframework/go-tuf/client"
	leveldbstore "github.com/theupdateframework/go-tuf/client/leveldbstore"
	"github.com/theupdateframework/go-tuf/data"

	"github.com/werf/lockgate"
	"github.com/werf/lockgate/pkg/file_locker"
	"github.com/werf/trdl/client/pkg/util"
)

const metaLocalStoreDirLockName = "meta"

type Client struct {
	*tufClient.Client

	memoryStore tufClient.LocalStore

	repoUrl           string
	metaLocalStoreDir string
	locker            lockgate.Locker
}

func NewClient(repoUrl, metaLocalStoreDir, locksPath string) (*Client, error) {
	c := &Client{}
	c.metaLocalStoreDir = metaLocalStoreDir
	c.repoUrl = repoUrl

	if err := c.initFileLocker(locksPath); err != nil {
		return nil, fmt.Errorf("unable to init file locker: %w", err)
	}

	if err := lockgate.WithAcquire(
		c.locker,
		metaLocalStoreDirLockName,
		lockgate.AcquireOptions{Shared: false, Timeout: time.Minute * 2},
		func(_ bool) error {
			if err := c.initTufClient(); err != nil {
				return err
			}

			return nil
		},
	); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Client) initFileLocker(locksPath string) error {
	locker, err := file_locker.NewFileLocker(locksPath)
	if err != nil {
		return err
	}

	c.locker = locker

	return nil
}

func (c *Client) initTufClient() error {
	localMeta := map[string]json.RawMessage{}
	{
		exist, err := util.IsRegularFileExist(c.metaLocalFilePath())
		if err != nil {
			return fmt.Errorf("unable to check regular file %q existence: %w", c.metaLocalFilePath(), err)
		}

		if exist {
			f, err := os.Open(c.metaLocalFilePath())
			if err != nil {
				return fmt.Errorf("unable to open file %q: %w", c.metaLocalFilePath(), err)
			}
			defer func() { _ = f.Close() }()

			d, err := io.ReadAll(f)
			if err != nil {
				return fmt.Errorf("unable to read file %q: %w", c.metaLocalFilePath(), err)
			}

			if err := json.Unmarshal(d, &localMeta); err != nil {
				return fmt.Errorf("unable to unmarshall data: %w", err)
			}
		} else { // backward compatibility with previous versions with leveldb backend
			localDB, err := leveldbstore.FileLocalStore(c.metaLocalStoreDir)
			if err != nil {
				return fmt.Errorf("unable to init file local store: %w", err)
			}

			localDBMeta, err := localDB.GetMeta()
			if err != nil {
				return fmt.Errorf("unable to get meta from file local store: %w", err)
			}
			if err := localDB.Close(); err != nil {
				return fmt.Errorf("unable to close from file local store: %w", err)
			}

			localMeta = localDBMeta
		}
	}

	localMemory := tufClient.MemoryLocalStore()
	for name, meta := range localMeta {
		if err := localMemory.SetMeta(name, meta); err != nil {
			return fmt.Errorf("unable to set meta: %w", err)
		}
	}

	remote, err := tufClient.HTTPRemoteStore(c.repoUrl, nil, nil)
	if err != nil {
		return fmt.Errorf("unable to init http remote store: %w", err)
	}

	c.Client = tufClient.NewClient(localMemory, remote)
	c.memoryStore = localMemory

	return nil
}

func (c *Client) Setup(rootVersion int64, rootSha512 string) error {
	return lockgate.WithAcquire(
		c.locker, metaLocalStoreDirLockName,
		lockgate.AcquireOptions{Shared: false, Timeout: time.Minute * 2},
		func(_ bool) error {
			if err := c.setup(rootVersion, rootSha512); err != nil {
				return err
			}

			return nil
		},
	)
}

func (c *Client) setup(rootVersion int64, rootSha512 string) error {
	var rootBasename string
	if rootVersion == 0 {
		rootBasename = "root.json"
	} else {
		rootBasename = fmt.Sprintf("%d.root.json", rootVersion)
	}

	jsonData, err := c.DownloadMetaUnsafe(rootBasename, tufClient.DefaultRootDownloadLimit)
	if err != nil {
		return fmt.Errorf("unable to download %q: %w", rootBasename, err)
	}

	rootFileChecksum := util.Sha512Checksum(jsonData)
	if rootFileChecksum != rootSha512 {
		return fmt.Errorf("expected hash sum of the root file %q not matched", rootFileChecksum)
	}

	rootKeys, err := tuf.ParseRootKeys(jsonData)
	if err != nil {
		return fmt.Errorf("unable to parse root keys: %w", err)
	}

	if err := os.RemoveAll(c.metaLocalStoreDir); err != nil {
		return fmt.Errorf("unable to remove directory %q: %w", c.metaLocalStoreDir, err)
	}

	if err := c.initTufClient(); err != nil {
		return fmt.Errorf("unable to reinit tuf client: %w", err)
	}

	if err := c.Client.Init(rootKeys, len(rootKeys)); err != nil {
		return err
	}

	if err := c.saveMeta(); err != nil {
		return fmt.Errorf("unable to save tuf meta: %w", err)
	}

	return nil
}

func (c *Client) Update() error {
	if _, err := c.Client.Update(); err != nil && !tufClient.IsLatestSnapshot(err) {
		return fmt.Errorf("unable to update tuf meta: %w", err)
	}

	return lockgate.WithAcquire(
		c.locker, metaLocalStoreDirLockName,
		lockgate.AcquireOptions{Shared: false, Timeout: time.Minute * 2},
		func(_ bool) error {
			if err := c.saveMeta(); err != nil {
				return fmt.Errorf("unable to save tuf meta: %w", err)
			}

			return nil
		},
	)
}

func (c *Client) saveMeta() error {
	allMeta, err := c.memoryStore.GetMeta()
	if err != nil {
		return fmt.Errorf("unable to get meta: %w", err)
	}

	if err := os.MkdirAll(c.metaLocalStoreDir, os.ModePerm); err != nil {
		return fmt.Errorf("unable to mkdirAll: %w", err)
	}

	f, err := os.OpenFile(c.tmpMetaLocalFilePath(), os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		return fmt.Errorf("unable to open file %q: %w", c.tmpMetaLocalFilePath(), err)
	}
	defer func() { _ = f.Close() }()

	d, err := json.Marshal(allMeta)
	if err != nil {
		return fmt.Errorf("unable to marshal meta: %w", err)
	}

	if _, err := f.Write(d); err != nil {
		return fmt.Errorf("unable to write file %q: %w", c.tmpMetaLocalFilePath(), err)
	}

	if err := os.Rename(c.tmpMetaLocalFilePath(), c.metaLocalFilePath()); err != nil {
		return fmt.Errorf("unable to rename file %q to %q: %w", c.tmpMetaLocalFilePath(), c.metaLocalFilePath(), err)
	}

	return nil
}

func (c Client) GetTargets() (data.TargetFiles, error) {
	return c.Client.Targets()
}

func (c Client) metaLocalFilePath() string {
	return filepath.Join(c.metaLocalStoreDir, "meta.json")
}

func (c Client) tmpMetaLocalFilePath() string {
	return filepath.Join(c.metaLocalStoreDir, "meta.json.tmp")
}
