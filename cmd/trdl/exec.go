package main

import (
	"fmt"

	"github.com/spf13/cobra"

	trdlClient "github.com/werf/trdl/pkg/client"
	"github.com/werf/trdl/pkg/trdl"
)

type execCmdData struct {
	repoName           string
	group              string
	channel            string
	optionalBinaryName string
	optionalBinaryArgs []string
}

func execCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "exec REPO GROUP [CHANNEL] [BINARY_NAME] [--] [ARGS]",
		Short:                 "Exec the channel release binary",
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cobra.MinimumNArgs(2)(cmd, args); err != nil {
				PrintHelp(cmd)
				return err
			}

			cmdData, err := processExecArgs(cmd, args)
			if err != nil {
				PrintHelp(cmd)
				return err
			}

			c, err := trdlClient.NewClient(trdlHomeDirectory)
			if err != nil {
				return fmt.Errorf("unable to initialize trdl client: %s", err)
			}

			if err := c.ExecRepoChannelReleaseBin(
				cmdData.repoName, cmdData.group, cmdData.channel,
				cmdData.optionalBinaryName, cmdData.optionalBinaryArgs,
			); err != nil {
				return fmt.Errorf("unable to exec release bin: %s", err)
			}

			return nil
		},
	}

	return cmd
}

func processExecArgs(cmd *cobra.Command, args []string) (*execCmdData, error) {
	data := &execCmdData{}

	data.repoName = args[0]
	data.group = args[1]

	doubleDashInd := cmd.ArgsLenAtDash()
	doubleDashExist := cmd.ArgsLenAtDash() != -1

	restArgs := args[2:]
	if doubleDashExist {
		data.optionalBinaryArgs = args[doubleDashInd:]
		restArgs = args[2:doubleDashInd]
	}

	switch len(restArgs) {
	case 0:
		data.channel = trdl.ChannelStable
		return data, nil
	case 1:
		for _, c := range trdl.Channels {
			if c == restArgs[0] {
				data.channel = restArgs[0]
				return data, nil
			}
		}
		data.channel = trdl.ChannelStable

		data.optionalBinaryName = restArgs[0]
		return data, nil
	case 2:
		channel, err := ParseOptionalChannelValue(restArgs[0])
		if err != nil {
			return nil, err
		}

		data.channel = channel
		data.optionalBinaryName = restArgs[1]
		return data, nil
	default:
		return nil, fmt.Errorf("unexpected positional args format")
	}
}
