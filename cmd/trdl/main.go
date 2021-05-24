package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const trdlHomeDirectory = "~/.trdl"

func main() {
	if err := rootCmd().Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err.Error())
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
		updateCmd(),
		execCmd(),
		dirPathCmd(),
		binPathCmd(),
		listCmd(),
		versionCmd(),
	)

	return rootCmd
}
