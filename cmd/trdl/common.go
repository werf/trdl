package main

import (
	"fmt"
	"strings"

	"github.com/asaskevich/govalidator"

	"github.com/werf/trdl/pkg/trdl"
)

func ParseOptionalChannelValue(value string) (string, error) {
	if value != "" {
		if !govalidator.IsIn(value, trdl.Channels...) {
			return "", fmt.Errorf(
				"unable to parse argument \"CHANNEL\": unsupported channel specified (%q), use one of the following: \"%s\"",
				value, strings.Join(trdl.Channels, `", "`))
		}

		return value, nil
	} else {
		return trdl.ChannelStable, nil
	}
}
