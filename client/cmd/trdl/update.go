package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/werf/common-go/pkg/util"
	trdlClient "github.com/werf/trdl/client/pkg/client"
	"github.com/werf/trdl/client/pkg/trdl"
)

func updateCmd() *cobra.Command {
	var noSelfUpdate bool
	var autoclean bool
	var inBackground bool
	var backgroundStdoutFile string
	var backgroundStderrFile string

	cmd := &cobra.Command{
		Use:                   "update REPO GROUP [CHANNEL]",
		Short:                 "Update the software",
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
					return fmt.Errorf("unable to start update in background: %w", err)
				}

				return nil
			}

			if !noSelfUpdate {
				if err := c.DoSelfUpdate(autoclean); err != nil {
					_, _ = fmt.Fprintf(os.Stderr, "WARNING: Self-update failed: %s\n", err)
				}
			}

			if err := c.UpdateRepoChannel(repoName, group, optionalChannel, autoclean); err != nil {
				return err
			}

			return nil
		},
	}

	SetupNoSelfUpdate(cmd, &noSelfUpdate)
	setupAutoclean(cmd, &autoclean)
	setupInBackground(cmd, &inBackground)
	setupBackgroundStdoutFile(cmd, &backgroundStdoutFile)
	setupBackgroundStderrFile(cmd, &backgroundStderrFile)

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
		return fmt.Errorf("unable to start command %q: %w\n", command, err)
	}

	if err := cmd.Process.Release(); err != nil {
		return fmt.Errorf("unable to release process: %w\n", err)
	}

	return nil
}

func setupAutoclean(cmd *cobra.Command, autoclean *bool) {
	envKey := "TRDL_AUTOCLEAN"

	cmd.Flags().BoolVar(autoclean,
		"autoclean",
		util.GetBoolEnvironmentDefaultTrue(envKey),
		fmt.Sprintf("Erase old downloaded releases (default $%s or true)", envKey))
}

func setupInBackground(cmd *cobra.Command, inBackground *bool) {
	envKey := "TRDL_IN_BACKGROUND"

	cmd.Flags().BoolVar(inBackground,
		"in-background",
		util.GetBoolEnvironmentDefaultFalse(envKey),
		fmt.Sprintf("Perform update in background (default $%s or false)", envKey))
}

func setupBackgroundStdoutFile(cmd *cobra.Command, backgroundStdoutFile *string) {
	envKey := "TRDL_BACKGROUND_STDOUT_FILE"

	cmd.Flags().StringVarP(backgroundStdoutFile,
		"background-stdout-file",
		"",
		os.Getenv(envKey),
		fmt.Sprintf("Redirect the stdout of the background update to a file (default $%s or none)", envKey))
}

func setupBackgroundStderrFile(cmd *cobra.Command, backgroundStderrFile *string) {
	envKey := "TRDL_BACKGROUND_STDERR_FILE"

	cmd.Flags().StringVarP(backgroundStderrFile,
		"background-stderr-file",
		"",
		os.Getenv(envKey),
		fmt.Sprintf("Redirect the stderr of the background update to a file (default $%s or none)", envKey))
}
