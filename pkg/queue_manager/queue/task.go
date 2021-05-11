package queue

import (
	"bytes"
	"context"

	"github.com/werf/logboek"
)

type Task struct {
	ctx           context.Context
	ctxCancelFunc context.CancelFunc
	uuid          string
	action        func() error
	buff          *bytes.Buffer
}

func NewTask(taskContext context.Context, uuid string, action func(ctx context.Context) error) *Task {
	buff := bytes.NewBuffer([]byte{})
	loggerCtx := logboek.NewContext(taskContext, logboek.DefaultLogger().NewSubLogger(buff, buff))
	taskContext, taskCtxCancelFunc := context.WithCancel(loggerCtx)

	return &Task{
		ctx:           taskContext,
		ctxCancelFunc: taskCtxCancelFunc,
		uuid:          uuid,
		action: func() error {
			return action(taskContext)
		},
		buff: buff,
	}
}
