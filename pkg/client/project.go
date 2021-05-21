package client

import (
	"path/filepath"

	"github.com/theupdateframework/go-tuf/client"
	leveldbstore "github.com/theupdateframework/go-tuf/client/leveldbstore"
	"github.com/theupdateframework/go-tuf/data"
)

type ProjectClient struct {
	projectName string
	directory   string
	tufClient   *client.Client
}

func newAppClient(projectName, directory, repoUrl string) (ProjectClient, error) {
	c := ProjectClient{
		projectName: projectName,
		directory:   directory,
	}

	if err := c.init(repoUrl); err != nil {
		return c, err
	}

	return c, nil
}

func (c *ProjectClient) init(repoUrl string) error {
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

func (c ProjectClient) TufClient() *client.Client {
	return c.tufClient
}

func (c ProjectClient) getTargets() (data.TargetFiles, error) {
	if _, err := c.tufClient.Update(); err != nil && !client.IsLatestSnapshot(err) {
		return nil, err
	}

	return c.tufClient.Targets()
}

func (c ProjectClient) metaLocalStorePath() string {
	return filepath.Join(c.directory, ".meta")
}
