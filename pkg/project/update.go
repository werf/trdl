package project

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/theupdateframework/go-tuf/data"
	util2 "github.com/theupdateframework/go-tuf/util"

	"github.com/werf/lockgate"

	"github.com/werf/trdl/pkg/trdl"
	"github.com/werf/trdl/pkg/util"
)

func (c Client) UpdateChannel(group, channel string) error {
	return lockgate.WithAcquire(c.locker, c.groupChannelLockName(group, channel), lockgate.AcquireOptions{Shared: false, Timeout: trdl.DefaultLockerTimeout}, func(_ bool) error {
		if err := c.syncMeta(); err != nil {
			return err
		}

		if err := c.syncChannel(group, channel); err != nil {
			return err
		}

		if err := c.syncChannelRelease(group, channel); err != nil {
			return err
		}

		return nil
	})
}

func (c Client) syncChannel(group, channel string) error {
	targets, err := c.tufClient.Targets()
	if err != nil {
		return err
	}

	targetName := c.channelTargetName(group, channel)
	targetMeta, ok := targets[targetName]
	if !ok {
		return fmt.Errorf("channel not found in the repo (group: %q, channel: %q)", group, channel)
	}

	path := c.channelPath(group, channel)
	return c.syncFile(targetName, targetMeta, path)
}

func (c Client) syncChannelRelease(group, channel string) error {
	releaseName, err := c.channelRelease(group, channel)
	if err != nil {
		return fmt.Errorf("unable to get channel release: %s", err)
	}

	var targets data.TargetFiles
	releaseTargetPrefix := filepath.Join(targetsReleases, releaseName)
	for _, prefix := range []string{
		filepath.Join(releaseTargetPrefix, fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)),
		filepath.Join(releaseTargetPrefix, fmt.Sprintf("%s-any", runtime.GOOS)),
		filepath.Join(releaseTargetPrefix, fmt.Sprintf("any-%s", runtime.GOARCH)),
		filepath.Join(releaseTargetPrefix, "any-any"),
	} {
		targets, err = c.filterTargets(prefix + "/")
		if err != nil {
			return err
		}

		if len(targets) != 0 {
			break
		}
	}

	if len(targets) == 0 {
		return fmt.Errorf(
			"nothing found in the repo for group: %q channel: %q os: %q arch: %q (release: %q)",
			group, channel, runtime.GOOS, runtime.GOARCH, releaseName,
		)
	}

	for name, meta := range targets {
		filePath := filepath.Join(c.directory, name)
		if err := c.syncFile(name, meta, filePath); err != nil {
			return err
		}
	}

	return nil
}

func (c Client) syncFile(targetName string, targetMeta data.TargetFileMeta, dest string) error {
	exist, err := util.IsRegularFileExist(dest)
	if err != nil {
		return fmt.Errorf("unable to check existence of file %q: %s", dest, err)
	}

	if exist {
		f, err := os.Open(dest)
		if err != nil {
			return fmt.Errorf("unable to open file %q, %s", dest, err)
		}
		defer func() {
			if err := f.Close(); err != nil {
				panic(err)
			}
		}()

		localFileMeta, err := util2.GenerateTargetFileMeta(f, targetMeta.FileMeta.HashAlgorithms()...)
		if err != nil {
			return fmt.Errorf("unable to generate meta for local file %q: %s", dest, err)
		}

		err = util2.TargetFileMetaEqual(targetMeta, localFileMeta)

		// file is up-to-date
		if err == nil {
			return nil
		}
	}

	return c.downloadFile(targetName, dest)
}

func (c Client) downloadFile(targetName string, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), os.ModePerm); err != nil {
		return err
	}

	f, err := os.OpenFile(dest, os.O_RDWR|os.O_CREATE, 0655)
	if err != nil {
		return err
	}
	file := destinationFile{f}

	if err := c.tufClient.Download(targetName, &file); err != nil {
		return err
	}

	return nil
}

func (c Client) filterTargets(prefix string) (data.TargetFiles, error) {
	targets, err := c.tufClient.Targets()
	if err != nil {
		return nil, err
	}

	result := data.TargetFiles{}
	for name, meta := range targets {
		if strings.HasPrefix(name, prefix) {
			result[name] = meta
		}
	}

	return result, nil
}
