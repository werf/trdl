package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/werf/trdl/release/cmd/trdl-vault/commands"
)

func main() {
	cmd := &cobra.Command{
		Use:   "trdl-vault",
		Short: "Trdl CLI for Vault operations",
	}

	cmd.AddCommand(commands.CreateCommands()...)
	cmd.SilenceUsage = true

	if err := cmd.Execute(); err != nil {
		fmt.Errorf("Execution failed: %s", err)
		os.Exit(1)
	}
}
