package client

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/werf/lockgate"
	"github.com/werf/lockgate/pkg/file_locker"

	"github.com/werf/trdl/pkg/repo"
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

func (c Client) AddRepo(repoName, repoUrl string, rootVersion int64, rootSha512 string) error {
	return lockgate.WithAcquire(c.locker, c.configurationPath(), lockgate.AcquireOptions{Shared: false, Timeout: trdl.DefaultLockerTimeout}, func(_ bool) error {
		if err := c.configuration.Reload(); err != nil {
			return err
		}

		c.configuration.StageRepoConfiguration(repoName, repoUrl)

		repoClient, err := c.GetRepoClient(repoName)
		if err != nil {
			return err
		}

		if err := repoClient.Init(repoUrl, rootVersion, rootSha512); err != nil {
			return fmt.Errorf("unable to init repository %q client: %s", repoName, err)
		}

		if err := c.configuration.Save(c.configurationPath()); err != nil {
			return fmt.Errorf("unable to save trdl configuration: %s", err)
		}

		return nil
	})
}

func (c Client) UpdateRepoChannel(repoName, group, channel string) error {
	repoClient, err := c.GetRepoClient(repoName)
	if err != nil {
		return err
	}

	return repoClient.UpdateChannel(group, channel)
}

func (c Client) ExecRepoChannelReleaseBin(repoName, group, channel string, optionalBinName string, args []string) error {
	repoClient, err := c.GetRepoClient(repoName)
	if err != nil {
		return err
	}

	if err := repoClient.ExecChannelReleaseBin(group, channel, optionalBinName, args); err != nil {
		switch e := err.(type) {
		case repo.ErrChannelNotFoundLocally:
			return prepareChannelNotFoundLocallyError(e)
		case repo.ErrChannelReleaseNotFoundLocally:
			return prepareChannelReleaseNotFoundLocallyError(e)
		case repo.ErrChannelReleaseBinSeveralFilesFound:
			return prepareChannelReleaseBinSeveralFilesFoundError(e)
		}

		return err
	}

	return nil
}

func (c Client) GetRepoChannelReleaseDir(repoName, group, channel string) (string, error) {
	repoClient, err := c.GetRepoClient(repoName)
	if err != nil {
		return "", err
	}

	dir, err := repoClient.GetChannelReleaseDir(group, channel)
	if err != nil {
		switch e := err.(type) {
		case repo.ErrChannelNotFoundLocally:
			return "", prepareChannelNotFoundLocallyError(e)
		case repo.ErrChannelReleaseNotFoundLocally:
			return "", prepareChannelReleaseNotFoundLocallyError(e)
		}

		return "", err
	}

	return dir, nil
}

func (c Client) GetRepoChannelReleaseBinDir(repoName, group, channel string) (string, error) {
	repoClient, err := c.GetRepoClient(repoName)
	if err != nil {
		return "", err
	}

	dir, err := repoClient.GetChannelReleaseBinDir(group, channel)
	if err != nil {
		switch e := err.(type) {
		case repo.ErrChannelNotFoundLocally:
			return "", prepareChannelNotFoundLocallyError(e)
		case repo.ErrChannelReleaseNotFoundLocally:
			return "", prepareChannelReleaseNotFoundLocallyError(e)
		}

		return "", err
	}

	return dir, nil
}

func prepareChannelNotFoundLocallyError(e repo.ErrChannelNotFoundLocally) error {
	return fmt.Errorf(
		"%s, update channel with \"trdl update %s %s %s\" command",
		e.Error(),
		e.RepoName,
		e.Group,
		e.Channel,
	)
}

func prepareChannelReleaseNotFoundLocallyError(e repo.ErrChannelReleaseNotFoundLocally) error {
	return fmt.Errorf(
		"%s, update channel with \"trdl update %s %s %s\" command",
		e.Error(),
		e.RepoName,
		e.Group,
		e.Channel,
	)
}

func prepareChannelReleaseBinSeveralFilesFoundError(e repo.ErrChannelReleaseBinSeveralFilesFound) error {
	return fmt.Errorf(
		"%s: it is necessary to specify the certain name:\n - %s",
		e.Error(),
		strings.Join(e.Names, "\n - "),
	)
}

func (c Client) GetRepoList() []*RepoConfiguration {
	return c.configuration.GetRepoConfigurationList()
}

func (c Client) GetRepoClient(repoName string) (RepoInterface, error) {
	return c.repoClient(repoName)
}

func (c Client) repoClient(repoName string) (RepoInterface, error) {
	repoDir := c.repoDir(repoName)
	if err := os.MkdirAll(repoDir, os.ModePerm); err != nil {
		return nil, err
	}

	repoUrl, err := c.repoRemoteUrl(repoName)
	if err != nil {
		return nil, err
	}

	return repo.NewClient(repoName, repoDir, repoUrl, c.repoLocksDir(repoName), c.repoTmpDir(repoName))
}

func (c *Client) repoDir(repoName string) string {
	return filepath.Join(c.dir, "repositories", repoName)
}

func (c *Client) repoRemoteUrl(repoName string) (string, error) {
	repoConfiguration := c.configuration.GetRepoConfiguration(repoName)
	if repoConfiguration == nil {
		return "", fmt.Errorf("repository %q not initialized: configure it with \"trdl add\" command", repoName)
	}

	return repoConfiguration.Url, nil
}

func (c *Client) configurationPath() string {
	return filepath.Join(c.dir, configurationFileBasename)
}

func (c *Client) repoLocksDir(repoName string) string {
	return filepath.Join(c.locksDir(), "repositories", repoName)
}

func (c *Client) locksDir() string {
	return filepath.Join(c.dir, ".locks")
}

func (c *Client) repoTmpDir(repoName string) string {
	return filepath.Join(c.tmpDir(), "repositories", repoName)
}

func (c *Client) tmpDir() string {
	return filepath.Join(c.dir, ".tmp")
}
