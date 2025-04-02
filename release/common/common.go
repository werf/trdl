package common

import (
	"log/slog"
	"net/url"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/werf/trdl/release/pkg/logger"
)

func SetupVaultAddress(cmdData *CmdData, cmd *cobra.Command) {
	cmdData.VaultAddress = new(string)
	defaultValue := "http://localhost:8200"
	envValue := os.Getenv("VAULT_ADDR")
	if envValue != "" {
		defaultValue = envValue
	}

	_, err := url.ParseRequestURI(defaultValue)
	if err != nil {
		panic(err)
	}

	cmd.PersistentFlags().StringVarP(cmdData.VaultAddress, "vault-address", "", defaultValue, "Set vault address")
}

func SetupVaultToken(cmdData *CmdData, cmd *cobra.Command) {
	cmdData.VaultToken = new(string)
	defaultValue := "root"
	envValue := os.Getenv("VAULT_TOKEN")
	if envValue != "" {
		defaultValue = envValue
	}

	cmd.PersistentFlags().StringVarP(cmdData.VaultToken, "vault-token", "", defaultValue, "Set vault token")
}

func SetupRetry(cmdData *CmdData, cmd *cobra.Command) {
	cmdData.Retry = new(bool)
	cmd.PersistentFlags().BoolVarP(cmdData.Retry, "retry", "", true, "Set flag to enable/disable retries")
}

func SetupMaxAttemps(cmdData *CmdData, cmd *cobra.Command) {
	cmdData.MaxAttempts = new(int)
	cmd.PersistentFlags().IntVarP(cmdData.MaxAttempts, "max-attempts", "", 5, "Set max retries")
}

func SetupDelay(cmdData *CmdData, cmd *cobra.Command) {
	cmdData.Delay = new(time.Duration)
	cmd.PersistentFlags().DurationVarP(cmdData.Delay, "delay", "", 10*time.Second, "Set max delay between retries")
}

func SetupLogLevel(cmdData *CmdData, cmd *cobra.Command) {
	cmdData.LogLevel = new(string)
	cmd.PersistentFlags().StringVarP(cmdData.LogLevel, "log-level", "", "", "Set log level (debug, info, warn, error)")
}

func (c *CmdData) GetLogLevel() slog.Level {
	return logger.ParseLogLevel(*c.LogLevel)
}
