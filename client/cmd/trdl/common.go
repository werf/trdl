package main

import (
	"fmt"
	"strings"

	"github.com/asaskevich/govalidator"
	"github.com/spf13/cobra"

	"github.com/werf/common-go/pkg/util"
	"github.com/werf/trdl/client/pkg/trdl"
)

func ValidateChannel(channel string) error {
	if !govalidator.IsIn(channel, trdl.Channels...) {
		return fmt.Errorf(
			"unable to parse argument \"CHANNEL\": unsupported channel %q specified, use one of the following: \"%s\"",
			channel, strings.Join(trdl.Channels, `", "`))
	}

	return nil
}

func SetupNoSelfUpdate(cmd *cobra.Command, noSelfUpdate *bool) {
	envKey := "TRDL_NO_SELF_UPDATE"

	cmd.Flags().BoolVar(noSelfUpdate,
		"no-self-update",
		util.GetBoolEnvironmentDefaultFalse(envKey),
		fmt.Sprintf("Do not perform self-update (default $%s or false)", envKey))
}

func PrintHelp(cmd *cobra.Command) {
	_ = cmd.Help()
	fmt.Println()
}
