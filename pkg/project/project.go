package project

import (
	"path/filepath"

	"github.com/theupdateframework/go-tuf/client"
	leveldbstore "github.com/theupdateframework/go-tuf/client/leveldbstore"
)

type Client struct {
	projectName string
	directory   string
	tufClient   *client.Client
}

func NewClient(projectName, directory, repoUrl string) (Client, error) {
	c := Client{
		projectName: projectName,
		directory:   directory,
	}

	if err := c.init(repoUrl); err != nil {
		return c, err
	}

	return c, nil
}

func (c *Client) init(repoUrl string) error {
	local, err := leveldbstore.FileLocalStore(c.metaLocalStorePath())
	if err != nil {
		return err
	}

	remote, err := client.HTTPRemoteStore(repoUrl, nil, nil)
	if err != nil {
		return err
	}

	c.tufClient = client.NewClient(local, remote)

	return nil
}

func (c Client) metaLocalStorePath() string {
	return filepath.Join(c.directory, ".meta")
}
