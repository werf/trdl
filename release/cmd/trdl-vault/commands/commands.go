package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/werf/trdl/client/pkg/logger"
	"github.com/werf/trdl/release/common"
	"github.com/werf/trdl/release/pkg/client"
)

func CreateCommands(cmdData *common.CmdData) []*cobra.Command {
	return []*cobra.Command{
		createPublishCommand(cmdData),
		createReleaseCommand(cmdData),
	}
}

func createPublishCommand(cmdData *common.CmdData) *cobra.Command {
	publishCmd := &cobra.Command{
		Use:   "publish <project-name>",
		Short: "Publish operation",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			projectName := args[0]
			if err := publish(cmdData, projectName); err != nil {
				logger.GlobalLogger.Error(fmt.Sprintf("Publish failed: %s", err))
				os.Exit(1)
			}
		},
	}

	common.SetupCmdData(cmdData, publishCmd)
	return publishCmd
}

func createReleaseCommand(cmdData *common.CmdData) *cobra.Command {

	releaseCmd := &cobra.Command{
		Use:   "release <project-name> <git-tag>",
		Short: "Release operation",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			projectName, gitTag := args[0], args[1]
			if err := release(cmdData, projectName, gitTag); err != nil {
				logger.GlobalLogger.Error(fmt.Sprintf("Release failed: %s", err))
				os.Exit(1)
			}
		},
	}

	common.SetupCmdData(cmdData, releaseCmd)
	return releaseCmd
}

func publish(c *common.CmdData, projectName string) error {
	trdlClient, err := client.NewTrdlVaultClient(client.NewTrdlVaultClientOpts{
		Address:     *c.Address,
		Token:       *c.Token,
		Retry:       *c.Retry,
		MaxAttempts: *c.MaxAttempts,
		Delay:       *c.Delay,
		LogLevel:    *c.LogLevel,
		LogFormat:   *c.LogFormat,
		Logger:      logger.GlobalLogger.LoggerInterface,
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
		Address:     *c.Address,
		Token:       *c.Token,
		Retry:       *c.Retry,
		MaxAttempts: *c.MaxAttempts,
		Delay:       *c.Delay,
		LogLevel:    *c.LogLevel,
		LogFormat:   *c.LogFormat,
		Logger:      logger.GlobalLogger.LoggerInterface,
	})
	if err != nil {
		return fmt.Errorf("unable to create client: %w", err)
	}
	if err := trdlClient.Release(projectName, gitTag); err != nil {
		return fmt.Errorf("unable to release project: %w", err)
	}
	return nil
}
