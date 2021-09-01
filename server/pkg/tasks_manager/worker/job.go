package worker

import (
	"context"

	"github.com/werf/logboek"
)

type Job struct {
	taskUUID      string
	action        func() error
	ctx           context.Context
	ctxCancelFunc context.CancelFunc
	buff          *SafeBuffer
}

type Task struct {
	Context context.Context
	UUID    string
	Action  func(ctx context.Context) error
}

func newJob(task *Task) *Job {
	buff := NewSafeBuffer()
	loggerCtx := logboek.NewContext(task.Context, logboek.DefaultLogger().NewSubLogger(buff, buff))
	jobContext, jobCtxCancelFunc := context.WithCancel(loggerCtx)

	return &Job{
		ctx:           jobContext,
		ctxCancelFunc: jobCtxCancelFunc,
		taskUUID:      task.UUID,
		action:        func() error { return task.Action(jobContext) },
		buff:          buff,
	}
}

func (j *Job) Log() []byte {
	return j.buff.Bytes()
}
