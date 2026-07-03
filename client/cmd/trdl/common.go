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

// isVersionArg reports whether the given positional argument is an explicit
// release version. Versions are distinguished from a GROUP (a plain version
// number, e.g. "2") by the leading "v" prefix (e.g. "v2.72.0").
func isVersionArg(arg string) bool {
	if len(arg) < 2 || arg[0] != 'v' {
		return false
	}
	return arg[1] >= '0' && arg[1] <= '9'
}

// parseVersionArg strips the leading "v" prefix from an explicit version
// argument, returning the bare version passed to the downstream code.
func parseVersionArg(arg string) string {
	return arg[1:]
}

// validateVersionArgs enforces that an explicit VERSION argument is mutually
// exclusive with the positional GROUP/CHANNEL arguments. When VERSION is given
// (args[1] has a "v" prefix), only REPO and VERSION are allowed; otherwise the
// default RangeArgs(2,3) behavior applies.
func validateVersionArgs(cmd *cobra.Command, args []string) error {
	if len(args) >= 2 && isVersionArg(args[1]) {
		if len(args) > 2 {
			return fmt.Errorf("VERSION is mutually exclusive with GROUP and CHANNEL arguments")
		}
		return nil
	}

	return cobra.RangeArgs(2, 3)(cmd, args)
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
