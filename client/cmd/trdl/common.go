package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Masterminds/semver/v3"
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

var explicitVersionPrefixRegexp = regexp.MustCompile(`^v\d`)

// isVersionArg reports whether the given positional argument is an explicit
// release version rather than a GROUP. An argument is a VERSION when it has a
// leading "v" prefix followed by a digit (e.g. "v2.72.0") or is a semver
// constraint (e.g. ">=1.2.0", "1.2.*"). A plain semver number without the "v"
// prefix (e.g. "1.2") is a GROUP, as is any other name (e.g. "vendor").
func isVersionArg(arg string) bool {
	if explicitVersionPrefixRegexp.MatchString(arg) {
		return true
	}

	if _, err := semver.NewVersion(arg); err == nil {
		return false
	}

	if _, err := semver.NewConstraint(arg); err == nil {
		return true
	}

	return false
}

// validateVersionArgs enforces that when an explicit VERSION argument is given
// only REPO and VERSION are allowed; otherwise the default RangeArgs(2,3)
// behavior applies.
func validateVersionArgs(cmd *cobra.Command, args []string) error {
	if len(args) >= 2 && isVersionArg(args[1]) {
		if len(args) > 2 {
			return fmt.Errorf("in VERSION mode only REPO and VERSION arguments are allowed")
		}

		if _, err := semver.NewConstraint(args[1]); err != nil {
			return fmt.Errorf("validate version: %w", err)
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
