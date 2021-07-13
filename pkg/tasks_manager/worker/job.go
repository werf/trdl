package worker

import (
	"context"
	"errors"

	"github.com/werf/logboek"
)

var contextCanceledError = errors.New("context canceled")

type Job struct {
	taskUUID      string
	action        func() error
	ctx           context.Context
	ctxCancelFunc context.CancelFunc
	buff          *SafeBuffer
}

func newJob(task *Task) *Job {
	buff := NewSafeBuffer()
	loggerCtx := logboek.NewContext(task.Context, logboek.DefaultLogger().NewSubLogger(buff, buff))
	jobContext, jobCtxCancelFunc := context.WithCancel(loggerCtx)

	return &Job{
		ctx:           jobContext,
		ctxCancelFunc: jobCtxCancelFunc,
		taskUUID:      task.UUID,
		action:        wrapTaskAction(jobContext, task.Action),
		buff:          buff,
	}
}

func wrapTaskAction(jobContext context.Context, taskAction func(ctx context.Context) error) func() error {
	return func() error {
		errCh := make(chan error)

		go func() {
			errCh <- taskAction(jobContext)
		}()

		select {
		case <-jobContext.Done():
			return contextCanceledError
		case err := <-errCh:
			return err
		}
	}
}

func (j *Job) Log() []byte {
	return j.buff.Bytes()
}
