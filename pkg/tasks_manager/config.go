package tasks_manager

import "time"

const storageKeyConfiguration = "tasks_manager_configuration"

type configuration struct {
	TaskTimeout      time.Duration `json:"task_timeout"`
	TaskHistoryLimit int           `json:"task_history_limit"`
}
