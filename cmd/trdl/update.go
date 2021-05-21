package main

import (
	"fmt"
	"strings"

	"github.com/asaskevich/govalidator"
	"github.com/spf13/cobra"

	trdlClient "github.com/werf/trdl/pkg/client"
)

func updateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "update PROJECT_NAME GROUP [CHANNEL]",
		Short:                 "Update the project channel files",
		DisableFlagsInUseLine: true,
		Args:                  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectName := args[0]
			group := args[1]

			var channel string
			if len(args) == 3 {
				value := args[2]
				if !govalidator.IsIn(value, channels...) {
					return fmt.Errorf(
						"unable to parse argument \"CHANNEL\": unsupported channel specified (%q), use one of the following: \"%s\"",
						value, strings.Join(channels, `", "`))
				}

				channel = value
			} else {
				channel = channelStable
			}

			c, err := trdlClient.NewClient(trdlHomeDirectory)
			if err != nil {
				return fmt.Errorf("unable to initialize trdl client: %s", err)
			}

			if err := c.UpdateProjectChannel(projectName, group, channel); err != nil {
				return fmt.Errorf("unable to update the project: %s", err)
			}

			return nil
		},
	}

	return cmd
}
