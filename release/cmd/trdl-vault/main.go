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

	cmd := &cobra.Command{
		Use:   "trdl-vault",
		Short: "Trdl CLI for Vault operations",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			logger.GlobalLogger = logger.SetupGlobalLogger(logger.LoggerOptions{
				Level:     *cmdData.LogLevel,
				LogFormat: *cmdData.LogFormat,
			})
			return nil
		},
	}

	common.SetupLogFormat(cmdData, cmd)
	common.SetupLogLevel(cmdData, cmd)

	cmd.AddCommand(commands.CreateCommands(cmdData)...)
	cmd.SilenceUsage = true

	if err := cmd.Execute(); err != nil {
		if logger.GlobalLogger != nil {
			logger.GlobalLogger.Error(fmt.Sprintf("Execution failed: %s", err))
		} else {
			fmt.Fprintf(os.Stderr, "Execution failed: %s\n", err)
		}
		os.Exit(1)
	}
}
