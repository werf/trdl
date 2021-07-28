package tuf

import (
	"fmt"

	tufClient "github.com/theupdateframework/go-tuf/client"
	leveldbstore "github.com/theupdateframework/go-tuf/client/leveldbstore"
	"github.com/theupdateframework/go-tuf/data"
)

type client struct {
	*tufClient.Client
	ReadOnlyLocalStore tufClient.LocalStore
	metaLocalStoreDir  string
}

func NewClient(metaLocalStoreDir, repoUrl string) (Client, error) {
	localDB, err := leveldbstore.FileLocalStore(metaLocalStoreDir)
	if err != nil {
		return nil, fmt.Errorf("unable to init file local store: %s", err)
	}

	allMeta, err := localDB.GetMeta()
	if err != nil {
		return nil, fmt.Errorf("unable to get meta from file local store: %s", err)
	}
	if err := localDB.Close(); err != nil {
		return nil, fmt.Errorf("unable to close from file local store: %s", err)
	}

	localMemory := tufClient.MemoryLocalStore()
	for name, meta := range allMeta {
		if err := localMemory.SetMeta(name, meta); err != nil {
			return nil, fmt.Errorf("unable to set meta: %s", err)
		}
	}

	remote, err := tufClient.HTTPRemoteStore(repoUrl, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to init http remote store: %s", err)
	}

	tc := tufClient.NewClient(localMemory, remote)
	c := &client{
		Client:             tc,
		ReadOnlyLocalStore: localMemory,
		metaLocalStoreDir:  metaLocalStoreDir,
	}

	return c, nil
}

func (c client) Init(rootKeys []*data.Key, threshold int) error {
	if err := c.Client.Init(rootKeys, threshold); err != nil {
		return err
	}

	if err := c.saveMeta(); err != nil {
		return fmt.Errorf("unable to save tuf meta: %s", err)
	}

	return nil
}

func (c client) DownloadMetaUnsafe(name string, maxMetaSize int64) ([]byte, error) {
	return c.Client.DownloadMetaUnsafe(name, maxMetaSize)
}

func (c *client) Update() error {
	if _, err := c.Client.Update(); err != nil && !tufClient.IsLatestSnapshot(err) {
		return fmt.Errorf("unable to update tuf meta: %s", err)
	}

	if err := c.saveMeta(); err != nil {
		return fmt.Errorf("unable to save tuf meta: %s", err)
	}

	return nil
}

func (c *client) saveMeta() error {
	localDB, err := leveldbstore.FileLocalStore(c.metaLocalStoreDir)
	if err != nil {
		return fmt.Errorf("unable to init file local store: %s", err)
	}

	allMeta, err := c.ReadOnlyLocalStore.GetMeta()
	if err != nil {
		return fmt.Errorf("unable to get meta: %s", err)
	}

	for name, meta := range allMeta {
		if err := localDB.SetMeta(name, meta); err != nil {
			return fmt.Errorf("unable to set meta into file local store: %s", err)
		}
	}
	if err := localDB.Close(); err != nil {
		return fmt.Errorf("unable to close from file local store: %s", err)
	}

	return nil
}

func (c client) GetTargets() (data.TargetFiles, error) {
	return c.Client.Targets()
}
