package tasks_manager

import "time"

const storageKeyConfiguration = "tasks_manager_configuration"

type configuration struct {
	TaskTimeout      time.Duration `structs:"task_timeout" json:"task_timeout"`
	TaskHistoryLimit int           `structs:"task_history_limit" json:"task_history_limit"`
}
