package main

import (
	"fmt"

	"github.com/asaskevich/govalidator"
	"github.com/spf13/cobra"

	trdlClient "github.com/werf/trdl/pkg/client"
)

func addCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "add PROJECT_NAME REPO_URL ROOT_VERSION ROOT_SHA512",
		Short:                 "Initialize the project",
		DisableFlagsInUseLine: true,
		Args:                  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectName := args[0]
			url := args[1]
			rootVersionArg := args[2]
			rootSha512 := args[3]

			rootVersion, err := parseRootVersionArgument(rootVersionArg)
			if err != nil {
				return fmt.Errorf("unable to parse required argument \"ROOT_VERSION\": %s", err)
			}

			c, err := trdlClient.NewClient(trdlHomeDirectory)
			if err != nil {
				return fmt.Errorf("unable to initialize trdl client: %s", err)
			}

			if err := c.AddProject(projectName, url, rootVersion, rootSha512); err != nil {
				return fmt.Errorf("unable to add the project: %s", err)
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
