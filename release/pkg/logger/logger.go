package logger

import "log"

type ConsoleLogger struct{}

func (cl *ConsoleLogger) Log(taskID, msg string) {
	log.Printf("[%s] %s", taskID, msg)
}
