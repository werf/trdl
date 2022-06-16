package repo

import (
	"github.com/werf/lockgate"
	"github.com/werf/trdl/client/pkg/trdl"
)

func (c Client) GetChannelReleaseBinDir(group, channel string) (dir string, err error) {
	err = lockgate.WithAcquire(c.locker, c.channelLockName(group, channel), lockgate.AcquireOptions{Shared: true, Timeout: trdl.DefaultLockerTimeout}, func(_ bool) error {
		dir, _, err = c.findChannelReleaseBinDir(group, channel)
		return err
	})

	return
}

func (c Client) GetChannelReleaseBinPath(group, channel, optionalBinName string) (path string, err error) {
	err = lockgate.WithAcquire(c.locker, c.channelLockName(group, channel), lockgate.AcquireOptions{Shared: true, Timeout: trdl.DefaultLockerTimeout}, func(_ bool) error {
		path, err = c.findChannelReleaseBinPath(group, channel, optionalBinName)
		return err
	})

	return
}
