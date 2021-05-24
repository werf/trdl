package project

import (
	"github.com/werf/lockgate"

	"github.com/werf/trdl/pkg/trdl"
)

func (c Client) ChannelReleaseBinDir(group, channel string) (dir string, err error) {
	err = lockgate.WithAcquire(c.locker, c.groupChannelLockName(group, channel), lockgate.AcquireOptions{Shared: true, Timeout: trdl.DefaultLockerTimeout}, func(_ bool) error {
		dir, _, err = c.channelReleaseBinDir(group, channel)
		return err
	})

	return
}
