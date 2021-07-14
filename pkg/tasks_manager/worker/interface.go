package worker

import "context"

type Interface interface {
	Start()
	CancelRunningJobByTaskUUID(uuid string) bool
	HoldRunningJobByTaskUUID(uuid string, do func(job *Job)) bool
}

type TaskCallbacksInterface interface {
	TaskStartedCallback(ctx context.Context, uuid string)
	TaskFailedCallback(ctx context.Context, uuid string, log []byte, err error)
	TaskSucceededCallback(ctx context.Context, uuid string, log []byte)
}
