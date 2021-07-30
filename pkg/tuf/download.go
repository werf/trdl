package tuf

import (
	"fmt"
	"os"
	"path/filepath"

	tufClient "github.com/theupdateframework/go-tuf/client"
	tufUtil "github.com/theupdateframework/go-tuf/util"
)

func (c Client) DownloadFile(targetName string, dest string, destMode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dest), os.ModePerm); err != nil {
		return err
	}

	f, err := os.OpenFile(dest, os.O_RDWR|os.O_CREATE, destMode)
	if err != nil {
		return err
	}
	defer func() {
		if err := f.Close(); err != nil {
			panic(fmt.Errorf("unable to close file: %s", err))
		}
	}()

	file := destinationFile{f}
	if err := c.Download(targetName, &file); err != nil {
		return err
	}

	return nil
}

type destinationFile struct {
	*os.File
}

func (t *destinationFile) Delete() error {
	_ = t.Close()
	return os.Remove(t.Name())
}

func (c Client) Download(targetName string, destination tufClient.Destination) error {
	return c.Client.Download(tufUtil.NormalizeTarget(targetName), destination)
}

func (c Client) DownloadMetaUnsafe(targetName string, maxMetaSize int64) ([]byte, error) {
	return c.Client.DownloadMetaUnsafe(tufUtil.NormalizeTarget(targetName), maxMetaSize)
}
