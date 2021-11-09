package main

import (
	"fmt"
	"os"

	"github.com/gookit/color"
	"github.com/spf13/cobra"

	"github.com/werf/trdl/client/cmd/trdl/command"
)

var homeDir string

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
	}

	rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})
	SetupHomeDir(rootCmd)

	groups := &command.Groups{}
	*groups = append(*groups, command.Groups{
		{
			Message: "Configuration commands",
			Commands: []*cobra.Command{
				addCmd(),
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

	cmd.Flags().StringVarP(&homeDir, "home-dir", "", defaultHomeDir, "Set trdl home directory (default $TRDL_HOME_DIR or ~/.trdl)")
}
