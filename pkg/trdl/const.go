package trdl

import "time"

const (
	ChannelAlpha     = "alpha"
	ChannelBeta      = "beta"
	ChannelEA        = "ea"
	ChannelStable    = "stable"
	ChannelRockSolid = "rock-solid"
	DefaultChannel   = ChannelStable

	ShellUnix       = "unix"
	ShellCmd        = "cmd"
	ShellPowerShell = "pwsh"
)

var Channels = []string{
	ChannelAlpha,
	ChannelBeta,
	ChannelEA,
	ChannelStable,
	ChannelRockSolid,
}

var DefaultLockerTimeout = 30 * time.Second
