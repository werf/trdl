package commands

import (
	"context"
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/werf/trdl/release/common"
	"github.com/werf/trdl/release/pkg/client"
)

func CreateCommands() []*cobra.Command {
	return []*cobra.Command{
		createPublishCommand(),
		createReleaseCommand(),
	}
}

func createPublishCommand() *cobra.Command {
	cmdData := &common.CmdData{}

	publishCmd := &cobra.Command{
		Use:   "publish <project-name>",
		Short: "Publish operation",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			projectName := args[0]
			if err := publish(cmdData, projectName); err != nil {
				log.Fatalf("Publish failed: %s", err.Error())
			}
		},
	}

	common.SetupCmdData(cmdData, publishCmd)
	publishCmd.SetContext(common.WithCmdData(context.Background(), cmdData))
	return publishCmd
}

func createReleaseCommand() *cobra.Command {
	cmdData := &common.CmdData{}

	releaseCmd := &cobra.Command{
		Use:   "release <project-name> <git-tag>",
		Short: "Release operation",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			projectName, gitTag := args[0], args[1]
			if err := release(cmdData, projectName, gitTag); err != nil {
				log.Fatalf("Release failed: %s", err.Error())
			}
		},
	}

	common.SetupCmdData(cmdData, releaseCmd)
	releaseCmd.SetContext(common.WithCmdData(context.Background(), cmdData))
	return releaseCmd
}

func publish(c *common.CmdData, projectName string) error {
	trdlClient, err := client.NewTrdlVaultClient(client.NewTrdlVaultClientOpts{
		VaultAddress: *c.VaultAddress,
		VaultToken:   *c.VaultToken,
		Retry:        *c.Retry,
		MaxAttempts:  *c.MaxAttempts,
		Delay:        *c.Delay,
		LogLevel:     c.GetLogLevel(),
	})
	if err != nil {
		return fmt.Errorf("unable to create client: %w", err)
	}
	if err := trdlClient.Publish(projectName); err != nil {
		return fmt.Errorf("unable to publish project: %w", err)
	}
	return nil
}

func release(c *common.CmdData, projectName, gitTag string) error {
	trdlClient, err := client.NewTrdlVaultClient(client.NewTrdlVaultClientOpts{
		VaultAddress: *c.VaultAddress,
		VaultToken:   *c.VaultToken,
		Retry:        *c.Retry,
		MaxAttempts:  *c.MaxAttempts,
		Delay:        *c.Delay,
		LogLevel:     c.GetLogLevel(),
	})
	if err != nil {
		return fmt.Errorf("unable to create client: %w", err)
	}
	if err := trdlClient.Release(projectName, gitTag); err != nil {
		return fmt.Errorf("unable to release project: %w", err)
	}
	return nil
}
