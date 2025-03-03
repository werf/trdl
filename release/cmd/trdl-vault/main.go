package main

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/werf/trdl/release/common"
	"github.com/werf/trdl/release/pkg/logger"
	"github.com/werf/trdl/release/pkg/vault"
)

func newVaultClient(vaultAddress, vaultToken string, Retry bool, maxAttempts int, Delay time.Duration, log *logger.Logger) *vault.TrdlClient {
	return vault.NewTrdlClient(vaultAddress, vaultToken, log, Retry, maxAttempts, Delay)
}

func main() {
	var commonCmdData common.CmdData

	var cmd = &cobra.Command{
		Use:   "trdl-vault",
		Short: "Trdl CLI for Vault operations",
	}
	log := logger.NewLogger(slog.LevelInfo)

	var publishCmd = &cobra.Command{
		Use:   "publish <project-name>",
		Short: "Publish operation",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			projectName := args[0]
			client := newVaultClient(
				*commonCmdData.VaultAddress,
				*commonCmdData.VaultToken,
				*commonCmdData.Retry,
				*commonCmdData.MaxAttempts,
				*commonCmdData.Delay,
				log,
			)

			client.Publish(projectName)
		},
	}

	var releaseCmd = &cobra.Command{
		Use:   "release <project-name> <git-tag>",
		Short: "Release operation",
		Args:  cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			projectName := args[0]
			gitTag := args[1]
			client := newVaultClient(
				*commonCmdData.VaultAddress,
				*commonCmdData.VaultToken,
				*commonCmdData.Retry,
				*commonCmdData.MaxAttempts,
				*commonCmdData.Delay,
				log,
			)

			client.Release(projectName, gitTag)
		},
	}

	common.SetupVaultAddress(&commonCmdData, cmd)
	common.SetupVaultToken(&commonCmdData, cmd)
	common.SetupRetry(&commonCmdData, cmd)
	common.SetupMaxAttemps(&commonCmdData, cmd)
	common.SetupDelay(&commonCmdData, cmd)

	cmd.AddCommand(publishCmd)
	cmd.AddCommand(releaseCmd)

	if err := cmd.Execute(); err != nil {
		log.Error("", fmt.Sprintf("Command execution failed: %v", err))
		os.Exit(1)
	}
}
