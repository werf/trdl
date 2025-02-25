package common

import "time"

type CmdData struct {
	ProjectName  *string
	GitTag       *string
	VaultAddress *string
	VaultToken   *string
	Retry        *bool
	MaxAttempts  *int
	Delay        *time.Duration
}
