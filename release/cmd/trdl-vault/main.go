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

func newVaultClient(vaultAddress, vaultToken string, Retry bool, maxAttempts int, Delay time.Duration, log *logger.Logger) (*vault.TrdlClient, error) {
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
		Use:   "publish",
		Short: "Publish operation",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newVaultClient(
				*commonCmdData.VaultAddress,
				*commonCmdData.VaultToken,
				*commonCmdData.Retry,
				*commonCmdData.MaxAttempts,
				*commonCmdData.Delay,
				log,
			)
			if err != nil {
				log.Error("", fmt.Sprintf("Failed to create Vault client: %v", err))
				return err
			}

			err = client.Publish(*commonCmdData.ProjectName)
			if err != nil {
				log.Error("", fmt.Sprintf("Publish failed: %v", err))
				return err
			}

			log.Info("", "Publish completed successfully!")
			return nil
		},
	}

	var releaseCmd = &cobra.Command{
		Use:   "release <git-tag>",
		Short: "Release operation",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			gitTag := args[0]
			client, err := newVaultClient(
				*commonCmdData.VaultAddress,
				*commonCmdData.VaultToken,
				*commonCmdData.Retry,
				*commonCmdData.MaxAttempts,
				*commonCmdData.Delay,
				log,
			)
			if err != nil {
				log.Error("", fmt.Sprintf("Failed to create Vault client: %v", err))
				return err
			}

			err = client.Release(*commonCmdData.ProjectName, gitTag)
			if err != nil {
				log.Error("", fmt.Sprintf("Release failed: %v", err))
				return err
			}

			log.Info("", "Release completed successfully!")
			return nil
		},
	}

	common.SetupProjectName(&commonCmdData, cmd)
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
