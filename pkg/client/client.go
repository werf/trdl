package client

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/theupdateframework/go-tuf"
	"github.com/theupdateframework/go-tuf/client"

	"github.com/werf/lockgate"
	"github.com/werf/lockgate/pkg/file_locker"

	"github.com/werf/trdl/pkg/util"
)

var defaultLockerTimeout = 30 * time.Second

type Client struct {
	dir           string
	configuration configurationInterface
	locker        lockgate.Locker
}

func NewClient(dir string) (Interface, error) {
	resolvedPath, err := util.ExpandPath(dir)
	if err != nil {
		return nil, fmt.Errorf("unable to expand path %q, %s", dir, err)
	}

	c := Client{
		dir: resolvedPath,
	}

	if err := c.init(); err != nil {
		return c, err
	}

	return c, nil
}

func (c *Client) init() error {
	if err := os.MkdirAll(c.dir, os.ModePerm); err != nil {
		return err
	}

	if err := c.initFileLocker(); err != nil {
		return fmt.Errorf("unable to init file locker: %s", err)
	}

	if err := c.initConfiguration(); err != nil {
		return err
	}

	return nil
}

func (c *Client) initFileLocker() error {
	locker, err := file_locker.NewFileLocker(c.locksPath())
	if err != nil {
		return err
	}

	c.locker = locker

	return nil
}

func (c *Client) initConfiguration() error {
	return lockgate.WithAcquire(c.locker, c.configurationPath(), lockgate.AcquireOptions{Shared: true, Timeout: defaultLockerTimeout}, func(_ bool) error {
		configuration, err := newConfiguration(c.configurationPath())
		if err != nil {
			return err
		}

		c.configuration = &configuration

		return nil
	})
}

func (c Client) AddProject(projectName, repoUrl string, rootVersion int64, rootSha512 string) error {
	return lockgate.WithAcquire(c.locker, c.configurationPath(), lockgate.AcquireOptions{Shared: false, Timeout: defaultLockerTimeout}, func(_ bool) error {
		c.configuration.StageProjectConfiguration(projectName, repoUrl)

		projectClient, err := c.ProjectClient(projectName)
		if err != nil {
			return err
		}

		var rootBasename string
		if rootVersion == 0 {
			rootBasename = "root.json"
		} else {
			rootBasename = fmt.Sprintf("%d.root.json", rootVersion)
		}

		data, err := projectClient.TufClient().DownloadMetaUnsafe(rootBasename, client.DefaultRootDownloadLimit)
		if err != nil {
			return fmt.Errorf("unable to download %q: %s", rootBasename, err)
		}

		rootFileChecksum := util.Sha512Checksum(data)
		if rootFileChecksum != rootSha512 {
			return fmt.Errorf("expected hash sum of the root file not matched (%q != %q)", rootSha512, rootFileChecksum)
		}

		rootKeys, err := tuf.ParseRootKeys(data)
		if err != nil {
			return fmt.Errorf("unable to parse root keys: %s", err)
		}

		if err := projectClient.TufClient().Init(rootKeys, len(rootKeys)); err != nil {
			return fmt.Errorf("unable to init tuf client: %s", err)
		}

		if err := c.configuration.Save(c.configurationPath()); err != nil {
			return fmt.Errorf("unable to save trdl configuration: %s", err)
		}

		return nil
	})
}

func (c Client) ListProjects() []*ProjectConfiguration {
	return c.configuration.GetProjectConfigurations()
}

func (c Client) ProjectClient(projectName string) (ProjectInterface, error) {
	return c.projectClient(projectName)
}

func (c Client) projectClient(projectName string) (ProjectInterface, error) {
	projectDirectory := c.projectDirectory(projectName)
	if err := os.MkdirAll(projectDirectory, os.ModePerm); err != nil {
		return nil, err
	}

	repoUrl, err := c.projectRemoteUrl(projectName)
	if err != nil {
		return nil, err
	}

	return newAppClient(projectName, projectDirectory, repoUrl)
}

func (c *Client) projectDirectory(projectName string) string {
	return filepath.Join(c.dir, "projects", projectName)
}

func (c *Client) projectRemoteUrl(projectName string) (string, error) {
	projectConfiguration := c.configuration.GetProjectConfiguration(projectName)
	if projectConfiguration == nil {
		return "", fmt.Errorf("project %q not initialized: configure it with \"trdl add\" command", projectName)
	}

	return projectConfiguration.RepoUrl, nil
}

func (c *Client) configurationPath() string {
	return filepath.Join(c.dir, configurationFileBasename)
}

func (c *Client) locksPath() string {
	return filepath.Join(c.dir, ".locks")
}
