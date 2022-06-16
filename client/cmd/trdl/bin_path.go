package main

import (
	"fmt"

	"github.com/spf13/cobra"

	trdlClient "github.com/werf/trdl/client/pkg/client"
	"github.com/werf/trdl/client/pkg/trdl"
)

func binPathCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "bin-path REPO GROUP [CHANNEL]",
		Short:                 "Get the directory with software binaries",
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cobra.RangeArgs(2, 3)(cmd, args); err != nil {
				PrintHelp(cmd)
				return err
			}

			repoName := args[0]
			group := args[1]

			if repoName == trdl.SelfUpdateDefaultRepo {
				PrintHelp(cmd)
				return fmt.Errorf("reserved repository name %q cannot be used", trdl.SelfUpdateDefaultRepo)
			}

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
				return fmt.Errorf("unable to initialize trdl client: %w", err)
			}

			dir, err := c.GetRepoChannelReleaseBinDir(repoName, group, optionalChannel)
			if err != nil {
				return err
			}

			fmt.Println(dir)

			return nil
		},
	}

	return cmd
}
