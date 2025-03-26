package tuf

import (
	"fmt"
	"net/http"
	"os"
	"time"

	tufClient "github.com/theupdateframework/go-tuf/client"
	leveldbstore "github.com/theupdateframework/go-tuf/client/leveldbstore"
	"github.com/theupdateframework/go-tuf/data"

	"github.com/werf/lockgate"
	"github.com/werf/lockgate/pkg/file_locker"
	"github.com/werf/trdl/client/pkg/logger"
	"github.com/werf/trdl/client/pkg/util"
)

const metaLocalStoreDirLockName = "meta"

type Client struct {
	*tufClient.Client
	RemoteStore        tufClient.RemoteStore
	ReadOnlyLocalStore tufClient.LocalStore

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
	localDB, err := leveldbstore.FileLocalStore(c.metaLocalStoreDir)
	if err != nil {
		return fmt.Errorf("unable to init file local store: %w", err)
	}

	allMeta, err := localDB.GetMeta()
	if err != nil {
		return fmt.Errorf("unable to get meta from file local store: %w", err)
	}
	if err := localDB.Close(); err != nil {
		return fmt.Errorf("unable to close from file local store: %w", err)
	}

	localMemory := tufClient.MemoryLocalStore()
	for name, meta := range allMeta {
		if err := localMemory.SetMeta(name, meta); err != nil {
			return fmt.Errorf("unable to set meta: %w", err)
		}
	}

	var remote tufClient.RemoteStore
	httpClient := &http.Client{
		Transport: &TracingTransport{
			Transport: http.DefaultTransport,
			Logger:    *logger.GlobalLogger,
		},
	}
	remote, err = tufClient.HTTPRemoteStore(c.repoUrl, nil, httpClient)
	if err != nil {
		return fmt.Errorf("unable to init http remote store: %w", err)
	}

	c.Client = tufClient.NewClient(localMemory, remote)
	c.ReadOnlyLocalStore = localMemory
	c.RemoteStore = remote

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

	jsonData, err := c.DownloadMeta(rootBasename)
	if err != nil {
		return fmt.Errorf("unable to download %q: %w", rootBasename, err)
	}

	rootFileChecksum := util.Sha512Checksum(jsonData)
	if rootFileChecksum != rootSha512 {
		return fmt.Errorf("expected hash sum of the root file %q not matched", rootFileChecksum)
	}

	if err := os.RemoveAll(c.metaLocalStoreDir); err != nil {
		return fmt.Errorf("unable to remove directory %q: %w", c.metaLocalStoreDir, err)
	}

	if err := c.initTufClient(); err != nil {
		return fmt.Errorf("unable to reinit tuf client: %w", err)
	}

	if err := c.Client.Init(jsonData); err != nil {
		return err
	}

	if err := c.saveMeta(); err != nil {
		return fmt.Errorf("unable to save tuf meta: %w", err)
	}

	return nil
}

func (c *Client) Update() error {
	if _, err := c.Client.Update(); err != nil {
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
	localDB, err := leveldbstore.FileLocalStore(c.metaLocalStoreDir)
	if err != nil {
		return fmt.Errorf("unable to init file local store: %w", err)
	}

	allMeta, err := c.ReadOnlyLocalStore.GetMeta()
	if err != nil {
		return fmt.Errorf("unable to get meta: %w", err)
	}

	for name, meta := range allMeta {
		if err := localDB.SetMeta(name, meta); err != nil {
			return fmt.Errorf("unable to set meta into file local store: %w", err)
		}
	}
	if err := localDB.Close(); err != nil {
		return fmt.Errorf("unable to close from file local store: %w", err)
	}

	return nil
}

func (c Client) GetTargets() (data.TargetFiles, error) {
	return c.Client.Targets()
}
