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
	cmdData := &common.CmdData{}

	log := logger.NewSlogLogger(logger.LoggerOptions{
		Level:     *cmdData.LogLevel,
		LogFormat: *cmdData.LogFormat,
	})

	cmd := &cobra.Command{
		Use:   "trdl-vault",
		Short: "Trdl CLI for Vault operations",
	}

	common.SetupLogFormat(cmdData, cmd)
	common.SetupLogLevel(cmdData, cmd)

	cmd.AddCommand(commands.CreateCommands(cmdData, log)...)
	cmd.SilenceUsage = true

	if err := cmd.Execute(); err != nil {
		log.Error(fmt.Sprintf("Execution failed: %s", err))
		os.Exit(1)
	}
}
