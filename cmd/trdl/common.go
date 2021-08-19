package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/asaskevich/govalidator"
	"github.com/spf13/cobra"

	"github.com/werf/trdl/pkg/trdl"
)

func ValidateChannel(channel string) error {
	if !govalidator.IsIn(channel, trdl.Channels...) {
		return fmt.Errorf(
			"unable to parse argument \"CHANNEL\": unsupported channel %q specified, use one of the following: \"%s\"",
			channel, strings.Join(trdl.Channels, `", "`))
	}

	return nil
}

func GetBoolEnvironmentDefaultFalse(environmentName string) bool {
	switch os.Getenv(environmentName) {
	case "1", "true", "yes":
		return true
	default:
		return false
	}
}

func PrintHelp(cmd *cobra.Command) {
	_ = cmd.Help()
	fmt.Println()
}
