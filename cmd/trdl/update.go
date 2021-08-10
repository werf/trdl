package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	trdlClient "github.com/werf/trdl/pkg/client"
)

func updateCmd() *cobra.Command {
	var inBackground bool
	var backgroundOutputFile string

	cmd := &cobra.Command{
		Use:                   "update REPO GROUP [CHANNEL]",
		Short:                 "Update channel",
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

			if inBackground {
				trdlBinPath := os.Args[0]

				var backgroundUpdateArgs []string
				for _, arg := range os.Args[1:] {
					if arg == "--in-background" || strings.HasPrefix(arg, "--in-background=") {
						continue
					}

					backgroundUpdateArgs = append(backgroundUpdateArgs, arg)
				}

				if err := StartUpdateInBackground(trdlBinPath, backgroundUpdateArgs, backgroundOutputFile); err != nil {
					return fmt.Errorf("unable to start update in background: %s", err)
				}

				return nil
			}

			if err := c.UpdateRepoChannel(repoName, group, optionalChannel); err != nil {
				return fmt.Errorf("unable to update channel: %s", err)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&inBackground, "in-background", false, "Perform update in background")
	cmd.Flags().StringVarP(&backgroundOutputFile, "background-output-file", "", "", "Redirect the output of the background update to a file")

	return cmd
}

func StartUpdateInBackground(name string, args []string, backgroundOutputFile string) error {
	cmd := exec.Command(name, args...)

	if backgroundOutputFile != "" {
		f, err := os.Create(backgroundOutputFile)
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()

		cmd.Stdout = f
		cmd.Stderr = f
	}

	if err := cmd.Start(); err != nil {
		command := strings.Join(append([]string{name}, args...), " ")
		return fmt.Errorf("unable to start command %q: %s\n", command, err)
	}

	if err := cmd.Process.Release(); err != nil {
		return fmt.Errorf("unable to release process: %s\n", err)
	}

	return nil
}
