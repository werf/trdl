package common

import (
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"
)

func SetupAddress(cmdData *CmdData, cmd *cobra.Command) {
	cmdData.Address = new(string)
	defaultValue := getEnvOrDefault("TRDL_VAULT_ADDR", "http://localhost:8200")

	_, err := url.ParseRequestURI(defaultValue)
	if err != nil {
		panic(err)
	}

	cmd.Flags().StringVarP(cmdData.Address, "address", "", defaultValue, "Set vault address (env: TRDL_VAULT_ADDR)")
}

func SetupToken(cmdData *CmdData, cmd *cobra.Command) {
	cmdData.Token = new(string)
	defaultValue := getEnvOrDefault("TRDL_VAULT_TOKEN", "root")

	cmd.Flags().StringVarP(cmdData.Token, "token", "", defaultValue, "Set vault token (env: TRDL_VAULT_TOKEN)")
}

func SetupRetry(cmdData *CmdData, cmd *cobra.Command) {
	cmdData.Retry = new(bool)
	defaultValue := getBoolFromEnv("TRDL_VAULT_RETRY", true)
	cmd.Flags().BoolVarP(cmdData.Retry, "retry", "", defaultValue, "Enable/disable retries (env: TRDL_VAULT_RETRY)")
}

func SetupMaxAttemps(cmdData *CmdData, cmd *cobra.Command) {
	cmdData.MaxAttempts = new(int)
	defaultValue := getIntFromEnv("TRDL_VAULT_MAX_ATTEMPTS", 5)
	cmd.Flags().IntVarP(cmdData.MaxAttempts, "max-attempts", "", defaultValue, "Set max retries (env: TRDL_VAULT_MAX_ATTEMPTS)")
}

func SetupDelay(cmdData *CmdData, cmd *cobra.Command) {
	cmdData.Delay = new(time.Duration)
	defaultValue := getDurationFromEnv("TRDL_VAULT_DELAY", 10*time.Second)
	cmd.Flags().DurationVarP(cmdData.Delay, "delay", "", defaultValue, "Set delay between retries (env: TRDL_VAULT_DELAY)")
}

func SetupLogLevel(cmdData *CmdData, cmd *cobra.Command) {
	cmdData.LogLevel = new(string)
	defaultValue := getEnvOrDefault("TRDL_VAULT_LOG_LEVEL", "info")
	cmd.Flags().StringVarP(cmdData.LogLevel, "log-level", "", defaultValue, "Set log level (debug, info, warn, error) (env: TRDL_VAULT_LOG_LEVEL)")
}

func SetupLogFormat(cmdData *CmdData, cmd *cobra.Command) {
	cmdData.LogFormat = new(string)
	defaultValue := getEnvOrDefault("TRDL_VAULT_LOG_FORMAT", "json")
	cmd.Flags().StringVarP(cmdData.LogFormat, "log-format", "", defaultValue, "Set log format (text, json) (env: TRDL_VAULT_LOG_FORMAT)")
}

func SetupCmdData(cmdData *CmdData, cmd *cobra.Command) {
	SetupAddress(cmdData, cmd)
	SetupToken(cmdData, cmd)
	SetupRetry(cmdData, cmd)
	SetupMaxAttemps(cmdData, cmd)
	SetupDelay(cmdData, cmd)
	SetupLogLevel(cmdData, cmd)
	SetupLogFormat(cmdData, cmd)
}

func getEnvOrDefault(envVar, defaultValue string) string {
	if val, exists := os.LookupEnv(envVar); exists {
		return val
	}
	return defaultValue
}

func getBoolFromEnv(envVar string, defaultValue bool) bool {
	val, exists := os.LookupEnv(envVar)
	if !exists {
		return defaultValue
	}
	parsed, err := strconv.ParseBool(val)
	if err != nil {
		return defaultValue
	}
	return parsed
}

func getIntFromEnv(envVar string, defaultValue int) int {
	val, exists := os.LookupEnv(envVar)
	if !exists {
		return defaultValue
	}
	parsed, err := strconv.Atoi(val)
	if err != nil {
		return defaultValue
	}
	return parsed
}

func getDurationFromEnv(envVar string, defaultValue time.Duration) time.Duration {
	val, exists := os.LookupEnv(envVar)
	if !exists {
		return defaultValue
	}
	parsed, err := time.ParseDuration(val)
	if err != nil {
		return defaultValue
	}
	return parsed
}
