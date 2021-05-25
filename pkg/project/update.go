package project

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/theupdateframework/go-tuf/data"
	util2 "github.com/theupdateframework/go-tuf/util"

	"github.com/werf/lockgate"

	"github.com/werf/trdl/pkg/trdl"
	"github.com/werf/trdl/pkg/util"
)

var (
	fileModeExecutable os.FileMode = 0o755
	fileModeRegular    os.FileMode = 0o655
)

func (c Client) UpdateChannel(group, channel string) error {
	return lockgate.WithAcquire(c.locker, c.updateChannelLockName(group, channel), lockgate.AcquireOptions{Shared: false, Timeout: time.Minute * 5}, func(_ bool) error {
		if err := c.syncMeta(); err != nil {
			return err
		}

		var deferErr error // the error affects the defer function
		var channelUpToDate bool
		var release string
		channelPath := c.channelPath(group, channel)
		channelTmpPath := c.channelTmpPath(group, channel)
		{ // create tmp channel if channel is not up-to-date
			targets, err := c.tufClient.Targets()
			if err != nil {
				return err
			}

			targetName := c.channelTargetName(group, channel)
			targetMeta, ok := targets[targetName]
			if !ok {
				return fmt.Errorf("channel not found in the repo (group: %q, channel: %q)", group, channel)
			}

			channelUpToDate, err = isLocalFileUpToDate(channelPath, targetMeta)
			if err != nil {
				return fmt.Errorf("unable to compare the file %q to the target: %s", channelPath, err)
			}

			var updateChannelPath string
			if !channelUpToDate {
				if err = c.syncFile(targetName, targetMeta, channelTmpPath, fileModeRegular); err != nil {
					return err
				}
				defer func() {
					if deferErr != nil {
						if removeErr := os.RemoveAll(channelTmpPath); removeErr != nil {
							panic(fmt.Errorf("unable to remove %q: %s", channelTmpPath, removeErr))
						}
					}
				}()

				updateChannelPath = channelTmpPath
			} else {
				updateChannelPath = channelPath
			}

			release, deferErr = readChannelRelease(updateChannelPath)
			if deferErr != nil {
				return fmt.Errorf("unable to get channel release: %s", deferErr)
			}
		}

		if deferErr = c.syncChannelReleaseWithLock(release); deferErr != nil {
			return deferErr
		}

		{ // rename tmp channel to channel (optional)
			if !channelUpToDate {
				return lockgate.WithAcquire(c.locker, c.channelLockName(group, channel), lockgate.AcquireOptions{Shared: false, Timeout: trdl.DefaultLockerTimeout}, func(_ bool) error {
					if deferErr = os.MkdirAll(filepath.Dir(channelPath), os.ModePerm); deferErr != nil {
						return fmt.Errorf("unable to mkdir all %q: %s", channelPath, deferErr)
					}

					if deferErr = os.Rename(channelTmpPath, channelPath); deferErr != nil {
						return fmt.Errorf("unable to rename file %q to %q: %s", channelTmpPath, channelPath, deferErr)
					}

					return nil
				})
			}
		}

		return nil
	})
}

func (c Client) syncChannelReleaseWithLock(release string) error {
	return lockgate.WithAcquire(c.locker, c.updateReleaseLockName(release), lockgate.AcquireOptions{Shared: false, Timeout: time.Minute * 5}, func(_ bool) error {
		return c.syncChannelRelease(release)
	})
}

func (c Client) syncChannelRelease(release string) error {
	targets, osArch, err := c.selectAppropriateReleaseTargets(release)
	if err != nil {
		return err
	}

	releaseTargetNamePrefix := c.releaseTargetNamePrefix(release)
	releaseTargetNamePrefixWithOSArch := path.Join(releaseTargetNamePrefix, osArch)

	var deferErr error // the error affects the defer function
	releaseDir := c.channelReleaseTmpDir(release)
	releaseTmpDir := c.channelReleaseTmpDir(release)
	{ // stop updating if all release files are up-to-date
		releaseFilesUpToDate := true
		for targetName, targetMeta := range targets {
			releaseFileRelPath := filepath.FromSlash(strings.TrimLeft(targetName, releaseTargetNamePrefix+"/"))
			releaseFilePath := filepath.Join(releaseDir, releaseFileRelPath)

			equal, err := isLocalFileUpToDate(releaseFilePath, targetMeta)
			if err != nil {
				return fmt.Errorf("unable to compare local file %q with target %q: %s", releaseFilePath, targetMeta, err)
			}

			if !equal {
				releaseFilesUpToDate = false
				break
			}
		}

		if releaseFilesUpToDate {
			return nil
		}

		defer func() {
			if deferErr != nil {
				if err := os.RemoveAll(releaseTmpDir); err != nil {
					panic(fmt.Errorf("unable to remove %q: %s", releaseTmpDir, err))
				}
			}
		}()
	}

	for targetName, targetMeta := range targets {
		var releaseFilePathMode os.FileMode
		isBinTarget := strings.HasPrefix(targetName, path.Join(releaseTargetNamePrefixWithOSArch, "bin")+"/")
		if isBinTarget {
			releaseFilePathMode = fileModeExecutable
		} else {
			releaseFilePathMode = fileModeRegular
		}

		releaseFileRelPath := filepath.FromSlash(strings.TrimLeft(targetName, releaseTargetNamePrefix+"/"))
		releaseFilePath := filepath.Join(releaseTmpDir, releaseFileRelPath)
		if deferErr = c.syncFile(targetName, targetMeta, releaseFilePath, releaseFilePathMode); deferErr != nil {
			return fmt.Errorf("unable to sync file %q: %s", releaseFilePath, deferErr)
		}
	}

	if deferErr = os.MkdirAll(filepath.Dir(releaseDir), os.ModePerm); deferErr != nil {
		return fmt.Errorf("unable to mkdir all %q: %s", releaseDir, deferErr)
	}

	if deferErr = os.Rename(releaseTmpDir, releaseDir); deferErr != nil {
		return fmt.Errorf("unable to rename file %q to %q: %s", releaseTmpDir, releaseDir, deferErr)
	}

	return nil
}

func (c Client) selectAppropriateReleaseTargets(release string) (targets data.TargetFiles, resultOsArch string, err error) {
	releaseTargetNamePrefix := c.releaseTargetNamePrefix(release)
	for _, osArch := range []string{
		fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH),
		fmt.Sprintf("%s-any", runtime.GOOS),
		fmt.Sprintf("any-%s", runtime.GOARCH),
		"any-any",
	} {
		prefix := path.Join(releaseTargetNamePrefix, osArch)
		targets, err = c.filterTargets(prefix + "/")
		if err != nil {
			return nil, "", err
		}

		if len(targets) != 0 {
			resultOsArch = osArch
			break
		}
	}

	if len(targets) == 0 {
		return nil, "", fmt.Errorf(
			"nothing found in the repo for release: %q, os: %q, arch: %q",
			release, runtime.GOOS, runtime.GOARCH,
		)
	}

	return targets, resultOsArch, nil
}

func (c Client) syncFile(targetName string, targetMeta data.TargetFileMeta, dest string, destMode os.FileMode) error {
	actual, err := isLocalFileUpToDate(dest, targetMeta)
	if err != nil {
		return err
	}

	// file is up-to-date
	if actual {
		return nil
	}

	return c.downloadFile(targetName, dest, destMode)
}

func (c Client) downloadFile(targetName string, dest string, destMode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dest), os.ModePerm); err != nil {
		return err
	}

	f, err := os.OpenFile(dest, os.O_RDWR|os.O_CREATE, destMode)
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

func isLocalFileUpToDate(path string, targetMeta data.TargetFileMeta) (bool, error) {
	exist, err := util.IsRegularFileExist(path)
	if err != nil {
		return false, fmt.Errorf("unable to check existence of file %q: %s", path, err)
	}

	if !exist {
		return false, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return false, fmt.Errorf("unable to open file %q, %s", path, err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			panic(err)
		}
	}()

	localFileMeta, err := util2.GenerateTargetFileMeta(f, targetMeta.FileMeta.HashAlgorithms()...)
	if err != nil {
		return false, fmt.Errorf("unable to generate meta for local file %q: %s", path, err)
	}

	err = util2.TargetFileMetaEqual(targetMeta, localFileMeta)
	equal := err == nil

	return equal, nil
}
