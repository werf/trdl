package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/werf/trdl/client/pkg/logger"
	"github.com/werf/trdl/release/cmd/trdl-vault/commands"
	"github.com/werf/trdl/release/common"
)

func main() {
	var logLevel, logFormat string
	cmdData := &common.CmdData{}

	cmd := &cobra.Command{
		Use:   "trdl-vault",
		Short: "Trdl CLI for Vault operations",
	}

	common.SetupLogFormat(&logFormat, cmd)
	common.SetupLogLevel(&logLevel, cmd)

	// parse flags before cmd.execute
	if err := cmd.ParseFlags(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse flags: %v\n", err)
		os.Exit(1)
	}
	var err error
	logLevel, err = cmd.Flags().GetString("log-level")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get log-level flag: %v\n", err)
		os.Exit(1)
	}

	logFormat, err = cmd.Flags().GetString("log-format")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get log-format flag: %v\n", err)
		os.Exit(1)
	}

	log := logger.NewSlogLogger(logger.LoggerOptions{
		Level:     logLevel,
		LogFormat: logFormat,
	})
	cmd.AddCommand(commands.CreateCommands(cmdData, log)...)
	cmd.SilenceUsage = true

	if err := cmd.Execute(); err != nil {
		log.Error(fmt.Sprintf("Execution failed: %s", err))
		os.Exit(1)
	}
}
