package main

import (
	"fmt"
	"os"

	"github.com/gookit/color"
	"github.com/spf13/cobra"
)

const trdlHomeDirectory = "~/.trdl"

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
		updateCmd(),
		execCmd(),
		dirPathCmd(),
		binPathCmd(),
		listCmd(),
		versionCmd(),
	)

	return rootCmd
}
