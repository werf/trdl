package worker

import (
	"bytes"
	"context"

	"github.com/werf/logboek"
)

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
	taskContext, taskCtxCancelFunc := context.WithCancel(loggerCtx)

	return &Job{
		ctx:           taskContext,
		ctxCancelFunc: taskCtxCancelFunc,
		taskUUID:      task.UUID,
		action: func() error {
			return task.Action(taskContext)
		},
		buff: buff,
	}
}

func (t *Job) Log() []byte {
	return t.buff.Bytes()
}
