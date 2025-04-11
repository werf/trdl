package common

import "time"

type CmdData struct {
	Address     *string
	Token       *string
	Retry       *bool
	MaxAttempts *int
	Delay       *time.Duration
}
