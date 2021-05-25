package client

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/werf/lockgate"
	"github.com/werf/lockgate/pkg/file_locker"

	"github.com/werf/trdl/pkg/project"
	"github.com/werf/trdl/pkg/trdl"
	"github.com/werf/trdl/pkg/util"
)

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
	locker, err := file_locker.NewFileLocker(c.locksDir())
	if err != nil {
		return err
	}

	c.locker = locker

	return nil
}

func (c *Client) initConfiguration() error {
	return lockgate.WithAcquire(c.locker, c.configurationPath(), lockgate.AcquireOptions{Shared: true, Timeout: trdl.DefaultLockerTimeout}, func(_ bool) error {
		configuration, err := newConfiguration(c.configurationPath())
		if err != nil {
			return err
		}

		c.configuration = &configuration

		return nil
	})
}

func (c Client) AddProject(projectName, repoUrl string, rootVersion int64, rootSha512 string) error {
	return lockgate.WithAcquire(c.locker, c.configurationPath(), lockgate.AcquireOptions{Shared: false, Timeout: trdl.DefaultLockerTimeout}, func(_ bool) error {
		if err := c.configuration.Reload(); err != nil {
			return err
		}

		c.configuration.StageProjectConfiguration(projectName, repoUrl)

		projectClient, err := c.ProjectClient(projectName)
		if err != nil {
			return err
		}

		if err := projectClient.Init(repoUrl, rootVersion, rootSha512); err != nil {
			return fmt.Errorf("unable to init project %q client: %s", projectName, err)
		}

		if err := c.configuration.Save(c.configurationPath()); err != nil {
			return fmt.Errorf("unable to save trdl configuration: %s", err)
		}

		return nil
	})
}

func (c Client) UpdateProjectChannel(projectName, group, channel string) error {
	projectClient, err := c.ProjectClient(projectName)
	if err != nil {
		return err
	}

	return projectClient.UpdateChannel(group, channel)
}

func (c Client) ExecProjectChannelReleaseBin(projectName, group, channel string, optionalBinName string, args []string) error {
	projectClient, err := c.ProjectClient(projectName)
	if err != nil {
		return err
	}

	if err := projectClient.ExecChannelReleaseBin(group, channel, optionalBinName, args); err != nil {
		switch e := err.(type) {
		case project.ErrChannelNotFoundLocally:
			return prepareChannelNotFoundLocallyError(e)
		case project.ErrChannelReleaseNotFoundLocally:
			return prepareChannelReleaseNotFoundLocallyError(e)
		case project.ErrChannelReleaseBinSeveralFilesFound:
			return prepareChannelReleaseBinSeveralFilesFoundError(e)
		}

		return err
	}

	return nil
}

func (c Client) ProjectChannelReleaseDir(projectName, group, channel string) (string, error) {
	projectClient, err := c.ProjectClient(projectName)
	if err != nil {
		return "", err
	}

	dir, err := projectClient.ChannelReleaseDir(group, channel)
	if err != nil {
		switch e := err.(type) {
		case project.ErrChannelNotFoundLocally:
			return "", prepareChannelNotFoundLocallyError(e)
		case project.ErrChannelReleaseNotFoundLocally:
			return "", prepareChannelReleaseNotFoundLocallyError(e)
		}

		return "", err
	}

	return dir, nil
}

func (c Client) ProjectChannelReleaseBinDir(projectName, group, channel string) (string, error) {
	projectClient, err := c.ProjectClient(projectName)
	if err != nil {
		return "", err
	}

	dir, err := projectClient.ChannelReleaseBinDir(group, channel)
	if err != nil {
		switch e := err.(type) {
		case project.ErrChannelNotFoundLocally:
			return "", prepareChannelNotFoundLocallyError(e)
		case project.ErrChannelReleaseNotFoundLocally:
			return "", prepareChannelReleaseNotFoundLocallyError(e)
		}

		return "", err
	}

	return dir, nil
}

func prepareChannelNotFoundLocallyError(e project.ErrChannelNotFoundLocally) error {
	return fmt.Errorf(
		"%s, update channel with \"trdl update %s %s %s\" command",
		e.Error(),
		e.ProjectName,
		e.Group,
		e.Channel,
	)
}

func prepareChannelReleaseNotFoundLocallyError(e project.ErrChannelReleaseNotFoundLocally) error {
	return fmt.Errorf(
		"%s, update channel with \"trdl update %s %s %s\" command",
		e.Error(),
		e.ProjectName,
		e.Group,
		e.Channel,
	)
}

func prepareChannelReleaseBinSeveralFilesFoundError(e project.ErrChannelReleaseBinSeveralFilesFound) error {
	return fmt.Errorf(
		"%s: it is necessary to specify the certain name:\n - %s",
		e.Error(),
		strings.Join(e.Names, "\n - "),
	)
}

func (c Client) ListProjects() []*ProjectConfiguration {
	return c.configuration.GetProjectConfigurations()
}

func (c Client) ProjectClient(projectName string) (ProjectInterface, error) {
	return c.projectClient(projectName)
}

func (c Client) projectClient(projectName string) (ProjectInterface, error) {
	projectDir := c.projectDir(projectName)
	if err := os.MkdirAll(projectDir, os.ModePerm); err != nil {
		return nil, err
	}

	repoUrl, err := c.projectRemoteUrl(projectName)
	if err != nil {
		return nil, err
	}

	return project.NewClient(projectName, projectDir, repoUrl, c.projectLocksDir(projectName), c.projectTmpDir(projectName))
}

func (c *Client) projectDir(projectName string) string {
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

func (c *Client) projectLocksDir(projectName string) string {
	return filepath.Join(c.locksDir(), "projects", projectName)
}

func (c *Client) locksDir() string {
	return filepath.Join(c.dir, ".locks")
}

func (c *Client) projectTmpDir(projectName string) string {
	return filepath.Join(c.tmpDir(), "projects", projectName)
}

func (c *Client) tmpDir() string {
	return filepath.Join(c.dir, ".tmp")
}
