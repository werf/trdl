package worker

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/werf/logboek"
)

var contextCanceledError = errors.New("context canceled")

type Job struct {
	taskUUID      string
	action        func() error
	ctx           context.Context
	ctxCancelFunc context.CancelFunc
	buff          *bytes.Buffer
}

func newJob(task *Task) *Job {
	buff := bytes.NewBuffer([]byte{})
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
			defer func() {
				p := recover()
				if p == nil || fmt.Sprint(p) == "send on closed channel" {
					return
				}

				panic(p)
			}()

			errCh <- taskAction(jobContext)
		}()

		select {
		case <-jobContext.Done():
			close(errCh)
			return contextCanceledError
		case err := <-errCh:
			return err
		}
	}
}

func (j *Job) Log() []byte {
	return j.buff.Bytes()
}
