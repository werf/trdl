package main

import (
	"fmt"
	"os"

	"github.com/gookit/color"
	"github.com/spf13/cobra"

	"github.com/werf/trdl/release/cmd/trdl-vault/commands"
)

func main() {
	cmd := &cobra.Command{
		Use:   "trdl-vault",
		Short: "Trdl CLI for Vault operations",
	}

	cmd.AddCommand(commands.CreateCommands()...)
	cmd.SilenceErrors = true

	if err := cmd.Execute(); err != nil {
		msg := fmt.Sprintf("Error: %s", err.Error())
		_, _ = fmt.Fprintln(os.Stderr, color.Red.Sprint(msg))
		os.Exit(1)
	}
}
