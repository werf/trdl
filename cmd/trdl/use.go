package main

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	trdlClient "github.com/werf/trdl/pkg/client"
	"github.com/werf/trdl/pkg/repo"
	"github.com/werf/trdl/pkg/trdl"
)

func useCmd() *cobra.Command {
	var noSelfUpdate bool
	var shell string

	cmd := &cobra.Command{
		Use:                   "use REPO GROUP [CHANNEL]",
		Short:                 "Generate script to use channel binaries in the current shell session",
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

			switch shell {
			case trdl.ShellUnix, trdl.ShellPowerShell:
			default:
				PrintHelp(cmd)
				return fmt.Errorf("specified shell %q not supported", shell)
			}

			c, err := trdlClient.NewClient(homeDir)
			if err != nil {
				return fmt.Errorf("unable to initialize trdl client: %s", err)
			}

			scriptPath, err := c.UseRepoChannelReleaseBinDir(
				repoName,
				group,
				optionalChannel,
				shell,
				repo.UseSourceOptions{NoSelfUpdate: noSelfUpdate},
			)
			if err != nil {
				return err
			}

			fmt.Println(scriptPath)

			return nil
		},
	}

	defaultShell := trdl.ShellUnix
	if runtime.GOOS == "windows" {
		defaultShell = trdl.ShellPowerShell
	}

	cmd.Flags().BoolVar(&noSelfUpdate, "no-self-update", GetBoolEnvironmentDefaultFalse("TRDL_NO_SELF_UPDATE"), "Do not perform self-update")
	cmd.Flags().StringVar(&shell, "shell", defaultShell, "Select the shell for which to prepare the script. Supports 'pwsh' and 'unix' shells")

	return cmd
}
