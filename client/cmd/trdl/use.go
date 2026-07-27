package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"

	trdlClient "github.com/werf/trdl/client/pkg/client"
	"github.com/werf/trdl/client/pkg/repo"
	"github.com/werf/trdl/client/pkg/trdl"
)

func useCmd() *cobra.Command {
	var noSelfUpdate bool
	var shell string

	cmd := &cobra.Command{
		Use:   "use REPO GROUP [CHANNEL] | REPO VERSION",
		Short: "Generate a script to use the software binaries within a shell session",
		Long:  `Generate a script to update the software binaries in the background and use local ones within a shell session`,
		Example: `  # Source script in a shell
  $ . $(trdl use repo_name 1.2 ea)

  # Pin an exact version (prefix it with "v"; a plain number like "1.2" is a GROUP)
  $ . $(trdl use repo_name v1.2.3)

  # Pin a semver constraint (resolves to the greatest matching release)
  $ . $(trdl use repo_name '>=1.2.0')

  # Force script generation for a Unix shell on Windows
  $ trdl use repo_name 1.2 ea --shell unix
`,
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateVersionArgs(cmd, args); err != nil {
				PrintHelp(cmd)
				return err
			}

			repoName := args[0]

			if repoName == trdl.SelfUpdateDefaultRepo {
				PrintHelp(cmd)
				return fmt.Errorf("reserved repository name %q cannot be used", trdl.SelfUpdateDefaultRepo)
			}

			switch shell {
			case trdl.ShellUnix, trdl.ShellPowerShell:
			default:
				PrintHelp(cmd)
				return fmt.Errorf("specified shell %q not supported", shell)
			}

			c, err := trdlClient.NewClient(homeDir)
			if err != nil {
				return fmt.Errorf("unable to initialize trdl client: %w", err)
			}

			if isVersionArg(args[1]) {
				scriptPath, err := c.UseRepoReleaseBinDir(
					repoName,
					args[1],
					shell,
					repo.UseSourceOptions{NoSelfUpdate: noSelfUpdate},
				)
				if err != nil {
					return err
				}

				fmt.Println(scriptPath)

				return nil
			}

			group := args[1]

			var optionalChannel string
			if len(args) == 3 {
				optionalChannel = args[2]
				if err := ValidateChannel(optionalChannel); err != nil {
					PrintHelp(cmd)
					return err
				}
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

	SetupNoSelfUpdate(cmd, &noSelfUpdate)
	SetupShell(cmd, &shell)

	return cmd
}

func SetupShell(cmd *cobra.Command, shell *string) {
	cmd.Flags().StringVar(shell, "shell", defaultShell(), `Select the shell for which to prepare the script. 
Supports 'pwsh' and 'unix' shells (default $TRDL_SHELL, 'pwsh' for Windows or 'unix')`)
}

func defaultShell() string {
	envValue := os.Getenv("TRDL_SHELL")
	if envValue != "" {
		return envValue
	}

	if runtime.GOOS == "windows" {
		return trdl.ShellPowerShell
	}

	return trdl.ShellUnix
}
