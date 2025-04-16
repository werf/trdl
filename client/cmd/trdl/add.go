package main

import (
	"fmt"

	"github.com/asaskevich/govalidator"
	"github.com/spf13/cobra"

	trdlClient "github.com/werf/trdl/client/pkg/client"
	"github.com/werf/trdl/client/pkg/trdl"
)

func addCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "add REPO URL [ROOT_VERSION] [ROOT_SHA512]",
		Short:                 "Add a software repository",
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			addArgs, err := parseAddCommandArgs(cmd, args)
			if err != nil {
				return fmt.Errorf("unable to parse command arguments: %w", err)
			}

			if addArgs.repoName == trdl.SelfUpdateDefaultRepo {
				PrintHelp(cmd)
				return fmt.Errorf("reserved repository name %q cannot be used", trdl.SelfUpdateDefaultRepo)
			}

			rootVersion, err := parseRootVersionArgument(addArgs.rootVersionArg)
			if err != nil {
				PrintHelp(cmd)
				return fmt.Errorf("unable to parse required argument \"ROOT_VERSION\": %w", err)
			}

			c, err := trdlClient.NewClient(homeDir)
			if err != nil {
				return fmt.Errorf("unable to initialize trdl client: %w", err)
			}

			if err := c.AddRepo(addArgs.repoName, addArgs.repoUrl, rootVersion, addArgs.rootSha512); err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}

type addCommandArgs struct {
	repoName       string
	repoUrl        string
	rootVersionArg string
	rootSha512     string
}

func parseAddCommandArgs(cmd *cobra.Command, args []string) (*addCommandArgs, error) {
	if len(args) != 2 && len(args) != 4 {
		PrintHelp(cmd)
		return nil, fmt.Errorf("expected 2 or 4 arguments: REPO URL [ROOT_VERSION] [ROOT_SHA512], got %d", len(args))
	}

	repoName := args[0]
	repoUrl := args[1]

	var rootVersionArg, rootSha512 string

	if len(args) == 4 {
		rootVersionArg = args[2]
		rootSha512 = args[3]

		if !govalidator.IsNumeric(rootVersionArg) {
			PrintHelp(cmd)
			return nil, fmt.Errorf("ROOT_VERSION must be numeric: %q", rootVersionArg)
		}

		if rootSha512 == "" {
			PrintHelp(cmd)
			return nil, fmt.Errorf("ROOT_SHA512 must not be empty")
		}
	}

	return &addCommandArgs{
		repoName:       repoName,
		repoUrl:        repoUrl,
		rootVersionArg: rootVersionArg,
		rootSha512:     rootSha512,
	}, nil
}

func parseRootVersionArgument(arg string) (int64, error) {
	if arg == "" {
		return 0, nil
	}
	if !govalidator.IsNumeric(arg) {
		return 0, fmt.Errorf("value (%q) must be an integer", arg)
	}

	return govalidator.ToInt(arg)
}
