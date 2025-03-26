package main

import (
	"fmt"
	"os"

	"github.com/gookit/color"
	"github.com/spf13/cobra"

	"github.com/werf/common-go/pkg/util"
	"github.com/werf/trdl/client/cmd/trdl/command"
	"github.com/werf/trdl/client/pkg/logger"
)

var (
	homeDir string
	debug   bool
)

func main() {
	if err := rootCmd().Execute(); err != nil {
		msg := fmt.Sprintf("Error: %s", err.Error())
		_, _ = fmt.Fprintln(os.Stderr, color.Red.Sprint(msg))
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "trdl",
		Long:          "The universal package manager for delivering your software updates securely from a TUF repository (more details on https://trdl.dev)",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			logLevel := "info"
			if debug {
				logLevel = "debug"
			}
			logger.GlobalLogger = logger.SetupGlobalLogger(logger.LoggerOptions{
				Level:     logLevel,
				LogFormat: "text",
			})
		},
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
	}

	rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})
	SetupHomeDir(rootCmd)

	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", util.GetBoolEnvironmentDefaultFalse("TRDL_DEBUG"), "Enable debug output (default $TRDL_DEBUG or false)")

	groups := &command.Groups{}
	*groups = append(*groups, command.Groups{
		{
			Message: "Configuration commands",
			Commands: []*cobra.Command{
				addCmd(),
				removeCmd(),
				listCmd(),
				setDefaultChannelCmd(),
			},
		},
		{
			Message: "Main commands",
			Commands: []*cobra.Command{
				useCmd(),
			},
		},
		{
			Message: "Advanced commands",
			Commands: []*cobra.Command{
				updateCmd(),
				execCmd(),
				dirPathCmd(),
				binPathCmd(),
				docsCmd(groups),
				versionCmd(),
			},
		},
	}...)
	groups.Add(rootCmd)

	command.ActsAsRootCommand(rootCmd, *groups...)

	return rootCmd
}

func SetupHomeDir(cmd *cobra.Command) {
	defaultHomeDir := os.Getenv("TRDL_HOME_DIR")
	if defaultHomeDir == "" {
		defaultHomeDir = "~/.trdl"
	}

	cmd.PersistentFlags().StringVarP(&homeDir, "home-dir", "", defaultHomeDir, "Set trdl home directory (default $TRDL_HOME_DIR or ~/.trdl)")
}
