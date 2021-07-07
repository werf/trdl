package worker

import "context"

type Interface interface {
	Start()
	Stop()
	HoldRunningJobByTaskUUID(uuid string, do func(job *Job)) bool
	HasRunningJobByTaskUUID(uuid string) bool
}

type Task struct {
	Context context.Context
	UUID    string
	Action  func(ctx context.Context) error
}
