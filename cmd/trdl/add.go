package main

import (
	"fmt"

	"github.com/asaskevich/govalidator"
	"github.com/spf13/cobra"

	trdlClient "github.com/werf/trdl/pkg/client"
)

func addCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "add REPO URL ROOT_VERSION ROOT_SHA512",
		Short:                 "Initialize the repository",
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cobra.ExactArgs(4)(cmd, args); err != nil {
				PrintHelp(cmd)
				return err
			}

			repoName := args[0]
			repoUrl := args[1]
			rootVersionArg := args[2]
			rootSha512 := args[3]

			rootVersion, err := parseRootVersionArgument(rootVersionArg)
			if err != nil {
				PrintHelp(cmd)
				return fmt.Errorf("unable to parse required argument \"ROOT_VERSION\": %s", err)
			}

			c, err := trdlClient.NewClient(trdlHomeDirectory)
			if err != nil {
				return fmt.Errorf("unable to initialize trdl client: %s", err)
			}

			if err := c.AddRepo(repoName, repoUrl, rootVersion, rootSha512); err != nil {
				return fmt.Errorf("unable to add the repository: %s", err)
			}

			return nil
		},
	}

	return cmd
}

func parseRootVersionArgument(arg string) (int64, error) {
	if !govalidator.IsNumeric(arg) {
		return 0, fmt.Errorf("value (%q) must be an integer", arg)
	}

	return govalidator.ToInt(arg)
}
