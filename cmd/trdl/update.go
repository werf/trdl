package main

import (
	"fmt"

	"github.com/spf13/cobra"

	trdlClient "github.com/werf/trdl/pkg/client"
)

func updateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "update PROJECT_NAME GROUP [CHANNEL]",
		Short:                 "Update the project channel files",
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cobra.RangeArgs(2, 3)(cmd, args); err != nil {
				PrintHelp(cmd)
				return err
			}

			projectName := args[0]
			group := args[1]

			var optionalChannelValue string
			if len(args) == 3 {
				optionalChannelValue = args[2]
			}

			channel, err := ParseOptionalChannelValue(optionalChannelValue)
			if err != nil {
				PrintHelp(cmd)
				return err
			}

			c, err := trdlClient.NewClient(trdlHomeDirectory)
			if err != nil {
				return fmt.Errorf("unable to initialize trdl client: %s", err)
			}

			if err := c.UpdateProjectChannel(projectName, group, channel); err != nil {
				return fmt.Errorf("unable to update channel: %s", err)
			}

			return nil
		},
	}

	return cmd
}
