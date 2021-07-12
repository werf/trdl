package worker

import "context"

type Interface interface {
	Start()
	CancelRunningJobByTaskUUID(uuid string) bool
	HoldRunningJobByTaskUUID(uuid string, do func(job *Job)) bool
}

type Task struct {
	Context context.Context
	UUID    string
	Action  func(ctx context.Context) error
}
