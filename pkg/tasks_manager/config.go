package tasks_manager

const storageKeyConfiguration = "tasks_manager_configuration"

type configuration struct {
	TaskTimeout      string `json:"task_timeout"`
	TaskHistoryLimit int    `json:"task_history_limit"`
}
