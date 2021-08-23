package util

import (
	"os"
)

func IsEnvVarTrue(envVarName string) bool {
	switch value := os.Getenv(envVarName); value {
	case "", "0", "false", "FALSE", "no", "NO":
		return false
	}
	return true
}
