package project

import (
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"
	"strings"

	"github.com/theupdateframework/go-tuf/client"
	leveldbstore "github.com/theupdateframework/go-tuf/client/leveldbstore"

	"github.com/werf/lockgate"
	"github.com/werf/lockgate/pkg/file_locker"

	"github.com/werf/trdl/pkg/util"
)

const (
	targetsChannels = "channels"
	targetsReleases = "releases"

	channelsDir = targetsChannels
	releasesDir = targetsReleases
)

type Client struct {
	projectName string
	directory   string
	tufClient   *client.Client
	locker      lockgate.Locker
}

func NewClient(projectName, directory, repoUrl, locksPath string) (Client, error) {
	c := Client{
		projectName: projectName,
		directory:   directory,
	}

	if err := c.init(repoUrl, locksPath); err != nil {
		return c, err
	}

	return c, nil
}

func (c *Client) init(repoUrl string, locksPath string) error {
	if err := c.initFileLocker(locksPath); err != nil {
		return err
	}

	if err := c.initTufClient(repoUrl); err != nil {
		return err
	}

	return nil
}

func (c *Client) initTufClient(repoUrl string) error {
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

func (c *Client) initFileLocker(locksPath string) error {
	locker, err := file_locker.NewFileLocker(locksPath)
	if err != nil {
		return err
	}

	c.locker = locker

	return nil
}

func (c Client) syncMeta() error {
	if _, err := c.tufClient.Update(); err != nil && !client.IsLatestSnapshot(err) {
		return err
	}

	return nil
}

func (c Client) channelTargetName(group, channel string) string {
	return path.Join(targetsChannels, group, channel)
}

func (c Client) channelPath(group, channel string) string {
	return filepath.Join(c.directory, channelsDir, group, channel)
}

func (c Client) channelRelease(group, channel string) (string, error) {
	channelFilePath := c.channelPath(group, channel)
	exist, err := util.IsRegularFileExist(channelFilePath)
	if err != nil {
		return "", fmt.Errorf("unable to check existence of file %q: %s", channelFilePath, err)
	}

	if !exist {
		return "", fmt.Errorf("chanel not found locally (group: %q, channel: %q)", group, channel)
	}

	channelData, err := ioutil.ReadFile(channelFilePath)
	if err != nil {
		return "", fmt.Errorf("unable to read file %q: %s", channelFilePath, err)
	}

	releaseName := strings.TrimSpace(string(channelData))
	return releaseName, nil
}

func (c Client) metaLocalStorePath() string {
	return filepath.Join(c.directory, ".meta")
}

func (c Client) groupChannelLockName(group, channel string) string {
	return fmt.Sprintf("%s-%s", group, channel)
}