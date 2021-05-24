package project

import (
	"fmt"
	"os"

	"github.com/theupdateframework/go-tuf"
	"github.com/theupdateframework/go-tuf/client"

	"github.com/werf/trdl/pkg/util"
)

func (c Client) Init(repoUrl string, rootVersion int64, rootSha512 string) error {
	if err := c.resetTufClient(repoUrl); err != nil {
		return err
	}

	var rootBasename string
	if rootVersion == 0 {
		rootBasename = "root.json"
	} else {
		rootBasename = fmt.Sprintf("%d.root.json", rootVersion)
	}

	jsonData, err := c.tufClient.DownloadMetaUnsafe(rootBasename, client.DefaultRootDownloadLimit)
	if err != nil {
		return fmt.Errorf("unable to download %q: %s", rootBasename, err)
	}

	rootFileChecksum := util.Sha512Checksum(jsonData)
	if rootFileChecksum != rootSha512 {
		return fmt.Errorf("expected hash sum of the root file %q not matched", rootFileChecksum)
	}

	rootKeys, err := tuf.ParseRootKeys(jsonData)
	if err != nil {
		return fmt.Errorf("unable to parse root keys: %s", err)
	}

	if err := c.tufClient.Init(rootKeys, len(rootKeys)); err != nil {
		return fmt.Errorf("unable to init tuf client: %s", err)
	}

	return nil
}

func (c *Client) resetTufClient(repoUrl string) error {
	if err := os.RemoveAll(c.metaLocalStoreDir()); err != nil {
		return err
	}

	return c.initTufClient(repoUrl)
}
