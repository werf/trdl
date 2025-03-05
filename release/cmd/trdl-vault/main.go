package main

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/werf/trdl/release/common"
	"github.com/werf/trdl/release/pkg/client"
)

func main() {
	var commonCmdData common.CmdData

	var cmd = &cobra.Command{
		Use:   "trdl-vault",
		Short: "Trdl CLI for Vault operations",
	}

	var publishCmd = &cobra.Command{
		Use:   "publish <project-name>",
		Short: "Publish operation",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			projectName := args[0]
			if err := publishCmd(&commonCmdData, projectName); err != nil {
				log.Fatalf("Publish failed: %s", err.Error())
			}
		},
	}

	var releaseCmd = &cobra.Command{
		Use:   "release <project-name> <git-tag>",
		Short: "Release operation",
		Args:  cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			projectName := args[0]
			gitTag := args[1]
			if err := releaseCmd(&commonCmdData, projectName, gitTag); err != nil {
				log.Fatalf("Release failed: %s", err.Error())
			}
		},
	}

	common.SetupVaultAddress(&commonCmdData, cmd)
	common.SetupVaultToken(&commonCmdData, cmd)
	common.SetupRetry(&commonCmdData, cmd)
	common.SetupMaxAttemps(&commonCmdData, cmd)
	common.SetupDelay(&commonCmdData, cmd)

	cmd.AddCommand(publishCmd)
	cmd.AddCommand(releaseCmd)

	cmd.SilenceUsage = true

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func publishCmd(c *common.CmdData, projectName string) error {
	trdlClient, err := client.NewTrdlVaultClient(client.NewTrdlVaultClientOpts{
		VaultAddress: *c.VaultAddress,
		VaultToken:   *c.VaultToken,
		Retry:        *c.Retry,
		MaxAttempts:  *c.MaxAttempts,
		Delay:        *c.Delay,
	})
	if err != nil {
		return fmt.Errorf("unable to create client: %w", err)
	}
	if err := trdlClient.Publish(projectName); err != nil {
		return fmt.Errorf("unable to publish project: %w", err)
	}
	return nil
}

func releaseCmd(c *common.CmdData, projectName, gitTag string) error {
	trdlClient, err := client.NewTrdlVaultClient(client.NewTrdlVaultClientOpts{
		VaultAddress: *c.VaultAddress,
		VaultToken:   *c.VaultToken,
		Retry:        *c.Retry,
		MaxAttempts:  *c.MaxAttempts,
		Delay:        *c.Delay,
	})
	if err != nil {
		return fmt.Errorf("unable to create client: %w", err)
	}
	if err := trdlClient.Release(projectName, gitTag); err != nil {
		return fmt.Errorf("unable to release project: %w", err)
	}
	return nil
}
