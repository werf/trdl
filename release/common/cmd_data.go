package common

import "time"

type CmdData struct {
	VaultAddress *string
	VaultToken   *string
	Retry        *bool
	MaxAttempts  *int
	Delay        *time.Duration
	LogLevel     *string
}
