package main

import (
	"fmt"

	"github.com/spf13/cobra"

	trdlClient "github.com/werf/trdl/client/pkg/client"
	"github.com/werf/trdl/client/pkg/trdl"
)

func removeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "remove REPO",
		Short:                 "Remove a software repository",
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cobra.ExactArgs(1)(cmd, args); err != nil {
				PrintHelp(cmd)
				return err
			}

			repoName := args[0]

			if repoName == trdl.SelfUpdateDefaultRepo {
				PrintHelp(cmd)
				return fmt.Errorf("reserved repository name %q cannot be used", trdl.SelfUpdateDefaultRepo)
			}

			c, err := trdlClient.NewClient(homeDir)
			if err != nil {
				return fmt.Errorf("unable to initialize trdl client: %w", err)
			}

			if err := c.RemoveRepo(repoName); err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}
