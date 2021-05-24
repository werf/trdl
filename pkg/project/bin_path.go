package project

import (
	"fmt"

	"github.com/werf/lockgate"

	"github.com/werf/trdl/pkg/trdl"
)

func (c Client) ChannelReleaseBinPath(group, channel string) (channelReleasePath string, err error) {
	err = lockgate.WithAcquire(c.locker, c.groupChannelLockName(group, channel), lockgate.AcquireOptions{Shared: true, Timeout: trdl.DefaultLockerTimeout}, func(_ bool) error {
		channelReleasePath, err = c.channelReleaseBinPath(group, channel)
		if err != nil {
			switch err.(type) {
			case ErrChannelNotFoundLocally, ErrChannelReleaseNotFoundLocally:
				return fmt.Errorf(
					"channel release files not found locally, update channel with \"trdl update %s %s %s\" command",
					c.projectName,
					group,
					channel,
				)
			}

			return err
		}

		return nil
	})

	return
}
