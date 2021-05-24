package project

import (
	"fmt"
	"strings"

	"github.com/werf/lockgate"

	"github.com/werf/trdl/pkg/trdl"
	"github.com/werf/trdl/pkg/util"
)

func (c Client) ExecChannelReleaseBin(group, channel string, optionalBinName string, args []string) error {
	return lockgate.WithAcquire(c.locker, c.groupChannelLockName(group, channel), lockgate.AcquireOptions{Shared: true, Timeout: trdl.DefaultLockerTimeout}, func(_ bool) error {
		path, err := c.channelReleaseBinPath(group, channel, optionalBinName)
		if err != nil {
			switch e := err.(type) {
			case ErrChannelNotFoundLocally, ErrChannelReleaseNotFoundLocally:
				return fmt.Errorf(
					"channel release files not found locally, update channel with \"trdl update %s %s %s\" command",
					c.projectName,
					group,
					channel,
				)
			case ErrChannelReleaseBinSeveralFilesFound:
				return fmt.Errorf(
					"%s: it is necessary to specify the certain name:\n - %s",
					err.Error(),
					strings.Join(e.names, "\n - "),
				)
			}

			return err
		}

		return util.Exec(path, args)
	})
}
