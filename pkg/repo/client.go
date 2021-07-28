package repo

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/werf/lockgate"
	"github.com/werf/lockgate/pkg/file_locker"

	"github.com/werf/trdl/pkg/tuf"
	"github.com/werf/trdl/pkg/util"
)

const (
	targetsChannels = "channels"
	targetsReleases = "releases"

	channelsDir = targetsChannels
	releasesDir = targetsReleases
)

type Client struct {
	repoName  string
	dir       string
	tpmDir    string
	tufClient tuf.Client
	locker    lockgate.Locker
}

func NewClient(repoName, dir, repoUrl, locksPath, tmpDir string) (Client, error) {
	c := Client{
		repoName: repoName,
		dir:      dir,
		tpmDir:   tmpDir,
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
		return fmt.Errorf("unable to init tuf client: %s", err)
	}

	return nil
}

func (c *Client) initTufClient(repoUrl string) (err error) {
	return lockgate.WithAcquire(c.locker, c.tufClientLockName(), lockgate.AcquireOptions{Shared: true, Timeout: time.Minute * 2}, func(_ bool) error {
		c.tufClient, err = tuf.NewClient(c.metaLocalStoreDir(), repoUrl)
		return err
	})
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
	return lockgate.WithAcquire(c.locker, c.tufClientLockName(), lockgate.AcquireOptions{Shared: false, Timeout: time.Minute * 2}, func(acquired bool) error {
		return c.tufClient.Update()
	})
}

func (c Client) channelTargetName(group, channel string) string {
	return path.Join(targetsChannels, group, channel)
}

func (c Client) releaseTargetNamePrefix(release string) string {
	return path.Join(targetsReleases, release)
}

func (c Client) channelPath(group, channel string) string {
	return filepath.Join(c.dir, channelsDir, group, channel)
}

func (c Client) channelReleaseDir(releaseName string) string {
	return filepath.Join(c.dir, releasesDir, releaseName)
}

func (c Client) channelTmpPath(group, channel string) string {
	return filepath.Join(c.tpmDir, channelsDir, group, channel)
}

func (c Client) channelReleaseTmpDir(releaseName string) string {
	return filepath.Join(c.tpmDir, releasesDir, releaseName)
}

func (c Client) findChannelReleaseBinPath(group, channel string, optionalBinName string) (string, error) {
	dir, releaseName, err := c.findChannelReleaseBinDir(group, channel)
	if err != nil {
		return "", err
	}

	var glob string
	if optionalBinName == "" {
		glob = filepath.Join(dir, "*")
	} else {
		glob = filepath.Join(dir, optionalBinName)
	}

	matches, err := filepath.Glob(glob)
	if err != nil {
		return "", fmt.Errorf("unable to glob files: %s", err)
	}

	if len(matches) > 1 {
		var names []string
		for _, m := range matches {
			names = append(names, strings.TrimPrefix(m, dir+string(os.PathSeparator)))
		}

		return "", NewChannelReleaseSeveralFilesFoundErr(c.repoName, group, channel, releaseName, names)
	} else if len(matches) == 0 {
		if optionalBinName == "" {
			return "", fmt.Errorf("binary file not found in release")
		} else {
			return "", fmt.Errorf("binary file %q not found in release", optionalBinName)
		}
	}

	return matches[0], nil
}

func (c Client) findChannelReleaseBinDir(group, channel string) (dir string, release string, err error) {
	releaseDir, releaseName, err := c.findChannelReleaseDir(group, channel)
	if err != nil {
		return "", "", err
	}

	binDir := filepath.Join(releaseDir, "bin")
	exist, err := util.IsDirExist(binDir)
	if err != nil {
		return "", "", fmt.Errorf("unable to check existence of directory %q: %s", binDir, err)
	}

	if !exist {
		return "", "", fmt.Errorf("bin directory not found in release directory (group: %q, channel: %q)", group, channel)
	}

	return binDir, releaseName, nil
}

func (c Client) findChannelReleaseDir(group, channel string) (dir string, release string, err error) {
	release, err = c.getChannelRelease(group, channel)
	if err != nil {
		return "", "", err
	}

	dirGlob := filepath.Join(c.dir, releasesDir, release, "*")

	matches, err := filepath.Glob(dirGlob)
	if err != nil {
		return "", "", fmt.Errorf("unable to glob files: %s", err)
	}

	if len(matches) > 1 {
		return "", "", fmt.Errorf("unexpected files in release directory:\n - %s", strings.Join(matches, "\n - "))
	} else if len(matches) == 0 {
		return "", "", NewChannelReleaseNotFoundLocallyErr(c.repoName, group, channel, release)
	}

	return matches[0], release, nil
}

func (c Client) getChannelRelease(group, channel string) (string, error) {
	channelFilePath := c.channelPath(group, channel)
	exist, err := util.IsRegularFileExist(channelFilePath)
	if err != nil {
		return "", fmt.Errorf("unable to check existence of file %q: %s", channelFilePath, err)
	}

	if !exist {
		return "", NewChannelNotFoundLocallyErr(c.repoName, group, channel)
	}

	return readChannelRelease(channelFilePath)
}

func readChannelRelease(path string) (string, error) {
	channelData, err := ioutil.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("unable to read file %q: %s", path, err)
	}

	releaseName := strings.TrimSpace(string(channelData))
	return releaseName, nil
}

func (c Client) metaLocalStoreDir() string {
	return filepath.Join(c.dir, ".meta")
}

func (c Client) channelLockName(group, channel string) string {
	return fmt.Sprintf("%s-%s", group, channel)
}

func (c Client) updateChannelLockName(group, channel string) string {
	return fmt.Sprintf("update-channel-%s-%s", group, channel)
}

func (c Client) updateReleaseLockName(release string) string {
	return fmt.Sprintf("update-release-%s", release)
}

func (c Client) tufClientLockName() string {
	return "tuf-client"
}
