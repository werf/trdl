package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const trdlHomeDirectory = "~/.trdl"

const (
	channelAlpha     = "alpha"
	channelBeta      = "beta"
	channelEA        = "ea"
	channelStable    = "stable"
	channelRockSolid = "rock-solid"
)

var channels = []string{
	channelAlpha,
	channelBeta,
	channelEA,
	channelStable,
	channelRockSolid,
}

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
		listCmd(),
		versionCmd(),
	)

	return rootCmd
}
