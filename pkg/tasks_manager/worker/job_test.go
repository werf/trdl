package worker

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// check that wrapTaskAction separates the processing of the context canceled and the execution of the action in the background
func TestWrapTaskAction(t *testing.T) {
	errChan := make(chan error)
	taskActionFinishedChan := make(chan bool)
	failTaskActionChan := make(chan error)
	ctx, ctxCancelFunc := context.WithCancel(context.Background())

	var taskActionErr error
	action := wrapTaskAction(ctx, func(ctx context.Context) error {
		taskActionErr = func() error {
			err := <-failTaskActionChan
			return err
		}()
		taskActionFinishedChan <- true
		return taskActionErr
	})

	// run wrapped action
	go func() {
		errChan <- action()
	}()

	// send context cancel
	ctxCancelFunc()

	// check context canceled
	assert.Equal(t, contextCanceledError, <-errChan)

	// stop task action with specific error
	expectedErr := fmt.Errorf("error")
	failTaskActionChan <- expectedErr

	// check task action finished
	<-taskActionFinishedChan
	assert.Equal(t, expectedErr, taskActionErr)
}
