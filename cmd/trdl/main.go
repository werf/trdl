package main

import (
	"fmt"
	"os"

	"github.com/gookit/color"
	"github.com/spf13/cobra"
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
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})

	rootCmd.AddCommand(
		addCmd(),
		setDefaultChannelCmd(),
		updateCmd(),
		execCmd(),
		dirPathCmd(),
		binPathCmd(),
		listCmd(),
		versionCmd(),
	)

	SetupHomeDir(rootCmd)

	return rootCmd
}

func SetupHomeDir(cmd *cobra.Command) {
	defaultHomeDir := os.Getenv("TRDL_HOME_DIR")
	if defaultHomeDir == "" {
		defaultHomeDir = "~/.trdl"
	}

	cmd.Flags().StringVarP(&homeDir, "home-dir", "", defaultHomeDir, "Set trdl home directory (default $TRDL_HOME_DIR or ~/.trdl)")
}
