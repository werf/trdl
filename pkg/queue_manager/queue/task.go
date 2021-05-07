package queue

import (
	"bytes"
	"context"

	"github.com/werf/logboek"
)

type task struct {
	ctx           context.Context
	ctxCancelFunc context.CancelFunc
	uuid          string
	action        func() error
	buff          *bytes.Buffer
}

func newTask(taskContext context.Context, uuid string, action func(ctx context.Context) error) *task {
	buff := bytes.NewBuffer([]byte{})
	loggerCtx := logboek.NewContext(taskContext, logboek.DefaultLogger().NewSubLogger(buff, buff))
	taskContext, taskCtxCancelFunc := context.WithCancel(loggerCtx)

	return &task{
		ctx:           taskContext,
		ctxCancelFunc: taskCtxCancelFunc,
		uuid:          uuid,
		action: func() error {
			return action(taskContext)
		},
		buff: buff,
	}
}
