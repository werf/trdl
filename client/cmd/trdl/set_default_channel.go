package main

import (
	"fmt"

	"github.com/spf13/cobra"

	trdlClient "github.com/werf/trdl/client/pkg/client"
)

func setDefaultChannelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-default-channel REPO CHANNEL",
		Short: "Set a default channel for a registered repository",
		Long: `Set a default channel for a registered repository.
The new channel will be used by default instead of stable`,
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cobra.ExactArgs(2)(cmd, args); err != nil {
				PrintHelp(cmd)
				return err
			}

			repoName := args[0]
			channel := args[1]

			if err := ValidateChannel(channel); err != nil {
				PrintHelp(cmd)
				return err
			}

			c, err := trdlClient.NewClient(homeDir)
			if err != nil {
				return fmt.Errorf("unable to initialize trdl client: %s", err)
			}

			if err := c.SetRepoDefaultChannel(repoName, channel); err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}
