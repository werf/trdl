package worker

import "context"

type Interface interface {
	Start()
	Stop()
	HoldRunningTask(uuid string, do func(job *Job)) bool
	HasRunningTaskByUUID(uuid string) bool
}

type Task struct {
	Context context.Context
	UUID    string
	Action  func(ctx context.Context) error
}
