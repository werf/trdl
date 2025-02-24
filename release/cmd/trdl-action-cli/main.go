package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/werf/trdl/release/pkg/vault"
)

type ConsoleLogger struct{}

func (cl *ConsoleLogger) Log(taskID, msg string) {
	log.Printf("[%s] %s", taskID, msg)
}

func newVaultClient(vaultAddr, vaultToken string, enableRetry bool, maxAttempts int, retryDelay time.Duration) (*vault.TrdlClient, error) {
	consoleLogger := &ConsoleLogger{}
	return vault.NewTrdlClient(vaultAddr, vaultToken, consoleLogger, enableRetry, maxAttempts, retryDelay)
}

func main() {
	var vaultToken string
	var vaultAddr string
	var projectName string
	var gitTag string
	var enableRetry bool
	var maxAttempts int
	var retryDelay time.Duration

	var rootCmd = &cobra.Command{
		Use:   "trdl-vault",
		Short: "Trdl CLI for Vault operations",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			requiredFlags := map[string]*string{
				"vault-addr":   &vaultAddr,
				"vault-token":  &vaultToken,
				"project-name": &projectName,
			}
			missingFlags := []string{}
			for flag, value := range requiredFlags {
				if *value == "" {
					missingFlags = append(missingFlags, flag)
				}
			}

			if len(missingFlags) > 0 {
				log.Fatalf("Error: required flags are missing: %v", missingFlags)
			}
		},
	}

	var publishCmd = &cobra.Command{
		Use:   "publish",
		Short: "Publish operation",
		Run: func(cmd *cobra.Command, args []string) {
			client, err := newVaultClient(vaultAddr, vaultToken, enableRetry, maxAttempts, retryDelay)
			if err != nil {
				log.Fatalf("Error initializing Vault client: %v", err)
			}

			log.Println("Starting publish...")
			err = client.Publish(projectName)
			if err != nil {
				log.Fatalf("Operation failed: %v", err)
			}

			log.Println("Publish completed successfully!")
		},
	}

	var releaseCmd = &cobra.Command{
		Use:   "release",
		Short: "Release operation",
		Run: func(cmd *cobra.Command, args []string) {
			client, err := newVaultClient(vaultAddr, vaultToken, enableRetry, maxAttempts, retryDelay)
			if err != nil {
				log.Fatalf("Error initializing Vault client: %v", err)
			}

			log.Println("Starting release...")
			err = client.Release(projectName, gitTag)
			if err != nil {
				log.Fatalf("Operation failed: %v", err)
			}

			log.Println("Release completed successfully!")
		},
	}

	rootCmd.PersistentFlags().StringVar(&vaultAddr, "vault-addr", os.Getenv("VAULT_ADDR"), "Vault address")
	rootCmd.PersistentFlags().StringVar(&vaultToken, "vault-token", os.Getenv("VAULT_TOKEN"), "Vault token")
	rootCmd.PersistentFlags().StringVar(&projectName, "project-name", os.Getenv("PROJECT_NAME"), "Project name")
	rootCmd.PersistentFlags().BoolVar(&enableRetry, "enable-retry", true, "Enable retries on failure")
	rootCmd.PersistentFlags().IntVar(&maxAttempts, "max-attempts", 5, "Maximum number of retry attempts")
	rootCmd.PersistentFlags().DurationVar(&retryDelay, "retry-delay", 10*time.Second, "Delay between retry attempts")

	releaseCmd.Flags().StringVar(&gitTag, "git-tag", "", "Git tag ")
	releaseCmd.MarkFlagRequired("git-tag")

	rootCmd.AddCommand(publishCmd)
	rootCmd.AddCommand(releaseCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
