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
	ShellPowerShell = "pwsh"

	SelfUpdateDefaultRepo        = "trdl"
	SelfUpdateDefaultUrl         = "https://tuf.trdl.dev"
	SelfUpdateDefaultRootVersion = 1
	SelfUpdateDefaultRootSha512  = "14e4127ef0fa3e34a6524eb6b540ff478766c5e5254b3687bbe8e727da2e748377f02f5c68d41c876990c7b6884656b55dd9992a555a35a76a6e2cdd23564501"
	SelfUpdateDefaultGroup       = "0"
)

var Channels = []string{
	ChannelAlpha,
	ChannelBeta,
	ChannelEA,
	ChannelStable,
	ChannelRockSolid,
}

var DefaultLockerTimeout = 30 * time.Second
