package common

import (
	"net/url"
	"os"
	"time"

	"github.com/spf13/cobra"
)

func SetupProjectName(cmdData *CmdData, cmd *cobra.Command) {
	cmdData.ProjectName = new(string)
	cmd.Flags().StringVarP(cmdData.ProjectName, "project-name", "N", os.Getenv("TRDL_PROJECT_NAME"), "Set a specific project name")
}

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

	cmd.Flags().StringVarP(cmdData.VaultAddress, "vault-address", "", defaultValue, "Set vault address")

}

func SetupVaultToken(cmdData *CmdData, cmd *cobra.Command) {
	cmdData.VaultToken = new(string)
	defaultValue := "root"
	envValue := os.Getenv("VAULT_TOKEN")
	if envValue != "" {
		defaultValue = envValue
	}

	cmd.Flags().StringVarP(cmdData.VaultToken, "vault-token", "", defaultValue, "Set vault token")

}

func SetupRetry(cmdData *CmdData, cmd *cobra.Command) {
	cmdData.Retry = new(bool)
	cmd.Flags().BoolVarP(cmdData.Retry, "retry", "", true, "Set flag to enable/disable retries")

}

func SetupMaxAttemps(cmdData *CmdData, cmd *cobra.Command) {
	cmdData.MaxAttempts = new(int)
	cmd.Flags().IntVarP(cmdData.MaxAttempts, "max-attemps", "", 5, "Set max retries")

}

func SetupDelay(cmdData *CmdData, cmd *cobra.Command) {
	cmdData.Delay = new(time.Duration)
	cmd.Flags().DurationVarP(cmdData.Delay, "delay", "", 10*time.Second, "Set max delay between retries")

}
