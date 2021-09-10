package util

import (
	"fmt"
)

type LogicalError error

func NewLogicalError(format string, a ...interface{}) LogicalError {
	return LogicalError(fmt.Errorf(format, a...))
}
