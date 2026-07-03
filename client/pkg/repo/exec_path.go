package repo

import (
	"github.com/werf/lockgate"
	"github.com/werf/trdl/client/pkg/trdl"
	"github.com/werf/trdl/client/pkg/util"
)

func (c Client) ExecChannelReleaseBin(group, channel, optionalBinName string, args []string) error {
	return lockgate.WithAcquire(c.locker, c.channelLockName(group, channel), lockgate.AcquireOptions{Shared: true, Timeout: trdl.DefaultLockerTimeout}, func(_ bool) error {
		path, err := c.findChannelReleaseBinPath(group, channel, optionalBinName)
		if err != nil {
			return err
		}

		return util.Exec(path, args)
	})
}

func (c Client) ExecReleaseBin(version, optionalBinName string, args []string) error {
	return lockgate.WithAcquire(c.locker, c.updateReleaseLockName(version), lockgate.AcquireOptions{Shared: true, Timeout: trdl.DefaultLockerTimeout}, func(_ bool) error {
		binDir, err := c.findReleaseBinDir(version)
		if err != nil {
			return err
		}

		path, err := c.findBinPathInDir(binDir, optionalBinName)
		if err != nil {
			if e, ok := err.(ReleaseBinSeveralFilesFoundError); ok {
				return NewReleaseBinSeveralFilesFoundError(c.repoName, version, e.Names)
			}
			return err
		}

		return util.Exec(path, args)
	})
}
