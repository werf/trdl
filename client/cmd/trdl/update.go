package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	trdlClient "github.com/werf/trdl/client/pkg/client"
	"github.com/werf/trdl/client/pkg/trdl"
)

func updateCmd() *cobra.Command {
	var noSelfUpdate bool
	var inBackground bool
	var backgroundStdoutFile string
	var backgroundStderrFile string

	cmd := &cobra.Command{
		Use:                   "update REPO GROUP [CHANNEL]",
		Short:                 "Update software",
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

				if err := StartUpdateInBackground(trdlBinPath, backgroundUpdateArgs, backgroundStdoutFile, backgroundStderrFile); err != nil {
					return fmt.Errorf("unable to start update in background: %s", err)
				}

				return nil
			}

			if !noSelfUpdate {
				if err := c.DoSelfUpdate(); err != nil {
					_, _ = fmt.Fprintf(os.Stderr, "WARNING: Self-update failed: %s\n", err)
				}
			}

			if err := c.UpdateRepoChannel(repoName, group, optionalChannel); err != nil {
				return err
			}

			return nil
		},
	}

	SetupNoSelfUpdate(cmd, &noSelfUpdate)
	cmd.Flags().BoolVar(&inBackground, "in-background", false, "Perform update in background")
	cmd.Flags().StringVarP(&backgroundStdoutFile, "background-stdout-file", "", "", "Redirect the stdout of the background update to a file")
	cmd.Flags().StringVarP(&backgroundStderrFile, "background-stderr-file", "", "", "Redirect the stderr of the background update to a file")

	return cmd
}

func StartUpdateInBackground(name string, args []string, backgroundStdoutFile, backgroundStderrFile string) error {
	cmd := exec.Command(name, args...)

	if backgroundStdoutFile != "" {
		f, err := os.Create(backgroundStdoutFile)
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()

		cmd.Stdout = f
	}

	if backgroundStderrFile != "" {
		f, err := os.Create(backgroundStderrFile)
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()

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
