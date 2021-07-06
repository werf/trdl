package queue_manager

const storageKeyConfiguration = "queue_manager_configuration"

type configuration struct {
	TaskTimeout      string `json:"task_timeout"`
	TaskHistoryLimit string `json:"task_history_limit"`
}
