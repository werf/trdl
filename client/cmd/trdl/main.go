package main

import (
	"fmt"
	"os"
	"time"

	"github.com/gookit/color"
	"github.com/spf13/cobra"

	"github.com/werf/trdl/client/cmd/trdl/command"
)

var homeDir string

func main() {
	if err := rootCmd().Execute(); err != nil {
		msg := fmt.Sprintf("Error: %s", err.Error())
		_, _ = fmt.Fprintln(os.Stderr, color.Red.Sprint(msg))
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "trdl",
		Long:          "The universal package manager for delivering your software updates securely from a TUF repository (more details on https://trdl.dev)",
		SilenceUsage:  true,
		SilenceErrors: true,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
	}

	rootCmd.SetHelpCommand(&cobra.Command{Hidden: true})
	SetupHomeDir(rootCmd)

	groups := &command.Groups{}
	*groups = append(*groups, command.Groups{
		{
			Message: "Configuration commands",
			Commands: []*cobra.Command{
				addCmd(),
				removeCmd(),
				listCmd(),
				setDefaultChannelCmd(),
			},
		},
		{
			Message: "Main commands",
			Commands: []*cobra.Command{
				useCmd(),
			},
		},
		{
			Message: "Advanced commands",
			Commands: []*cobra.Command{
				updateCmd(),
				execCmd(),
				dirPathCmd(),
				binPathCmd(),
				docsCmd(groups),
				versionCmd(),
			},
		},
	}...)
	groups.Add(rootCmd)

	command.ActsAsRootCommand(rootCmd, *groups...)
	
	for _, cmd := range rootCmd.Commands() {
		copyCmdRunE := cmd.RunE 
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			startTime := time.Now()
			
			log := func(event, format string, args ...interface{}) {
				fmt.Printf("[%s] [%.2fs] %s: %s\n",
					time.Now().Format("15:04:05.000"), time.Since(startTime).Seconds(), event, fmt.Sprintf(format, args...))
			}

			log("COMMAND_STARTED", cmd.Name())
			
			err := copyCmdRunE(cmd, args)
			
			log("COMMAND_DONE", "(Total: %.2fs)", time.Since(startTime).Seconds())
			
			return err
		}
	}

	return rootCmd
}

func SetupHomeDir(cmd *cobra.Command) {
	defaultHomeDir := os.Getenv("TRDL_HOME_DIR")
	if defaultHomeDir == "" {
		defaultHomeDir = "~/.trdl"
	}

	cmd.PersistentFlags().StringVarP(&homeDir, "home-dir", "", defaultHomeDir, "Set trdl home directory (default $TRDL_HOME_DIR or ~/.trdl)")
}
