package main

import (
	"fmt"

	"github.com/spf13/cobra"

	trdlClient "github.com/werf/trdl/pkg/client"
)

func dirPathCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "dir-path REPO GROUP [CHANNEL]",
		Short:                 "Get path to the directory with channel release files",
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cobra.RangeArgs(2, 3)(cmd, args); err != nil {
				PrintHelp(cmd)
				return err
			}

			repoName := args[0]
			group := args[1]

			var optionalChannel string
			if len(args) == 3 {
				optionalChannel = args[2]
				if err := ValidateChannel(optionalChannel); err != nil {
					PrintHelp(cmd)
					return err
				}
			}

			c, err := trdlClient.NewClient(homeDir)
			if err != nil {
				return fmt.Errorf("unable to initialize trdl client: %s", err)
			}

			dir, err := c.GetRepoChannelReleaseDir(repoName, group, optionalChannel)
			if err != nil {
				return err
			}

			fmt.Println(dir)

			return nil
		},
	}

	return cmd
}
