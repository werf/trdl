package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/werf/trdl/pkg/trdl"
)

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:                   "version",
		DisableFlagsInUseLine: true,
		Short:                 "Print version",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println(trdl.Version)
		},
	}
}
