package worker

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/werf/logboek"
)

const (
	TaskFailedCallback    = "FAILED"
	TaskCompletedCallback = "COMPLETED"
)

func TestTaskCallbacks(t *testing.T) {
	taskChan := make(chan *Task)
	taskProcessedChan := make(chan bool)

	var expectedUUID string
	var expectedCallback string
	var expectedErr error
	var expectedLog []byte

	w := NewWorker(taskChan, Callbacks{
		TaskStartedCallback: func(_ context.Context, uuid string) error {
			assert.Equal(t, expectedUUID, uuid)
			return nil
		},
		TaskFailedCallback: func(_ context.Context, uuid string, log []byte, err error) error {
			assert.Equal(t, expectedCallback, TaskFailedCallback)
			assert.Equal(t, expectedUUID, uuid)
			assert.Equal(t, expectedErr, err)
			assert.Equal(t, expectedLog, log)
			taskProcessedChan <- true
			return nil
		},
		TaskCompletedCallback: func(_ context.Context, uuid string, log []byte) error {
			assert.Equal(t, expectedCallback, TaskCompletedCallback)
			assert.Equal(t, uuid, expectedUUID)
			assert.Nil(t, expectedErr)
			assert.Equal(t, log, expectedLog)
			taskProcessedChan <- true
			return nil
		},
	})
	go w.Start()

	for _, c := range []struct {
		uuid     string
		callback string
		log      []byte
		err      error
	}{
		{
			uuid:     "1",
			callback: TaskCompletedCallback,
			log:      []byte("hello"),
			err:      nil,
		},
		{
			uuid:     "2",
			callback: TaskFailedCallback,
			log:      []byte("error"),
			err:      errors.New("error"),
		},
	} {
		expectedUUID = c.uuid
		expectedCallback = c.callback
		expectedLog = c.log
		expectedErr = c.err

		taskChan <- &Task{
			Context: context.Background(),
			UUID:    expectedUUID,
			Action: func(ctx context.Context) error {
				logboek.Context(ctx).Log(string(expectedLog))

				if expectedErr != nil {
					return expectedErr
				}

				return nil
			},
		}

		<-taskProcessedChan
	}
}

func TestWorker_Stop(t *testing.T) {
	taskChan := make(chan *Task)
	taskFailedChan := make(chan bool)
	workerStoppedChan := make(chan bool)
	taskUUID := "1"

	w := NewWorker(taskChan, Callbacks{
		TaskStartedCallback: func(_ context.Context, uuid string) error { return nil },
		TaskFailedCallback: func(_ context.Context, uuid string, _ []byte, err error) error {
			assert.Equal(t, taskUUID, uuid)
			assert.Equal(t, fmt.Errorf("context canceled"), err)
			taskFailedChan <- true
			return nil
		},
	})
	go func() {
		w.Start()

		workerStoppedChan <- true
	}()

	taskChan <- &Task{
		Context: context.Background(),
		UUID:    taskUUID,
		Action:  infiniteTask,
	}

	go w.Stop()

	// check task failed
	<-taskFailedChan

	// check worker stopped
	<-workerStoppedChan
}

func TestWorker_HasRunningJobByTaskUUID(t *testing.T) {
	taskChan := make(chan *Task)
	taskCompletedChan := make(chan bool)
	taskUUID := "1"

	w := NewWorker(taskChan, Callbacks{
		TaskStartedCallback: func(_ context.Context, uuid string) error { return nil },
		TaskCompletedCallback: func(ctx context.Context, uuid string, log []byte) error {
			taskCompletedChan <- true
			return nil
		},
	})
	go w.Start()

	// check when task not started yet
	assert.False(t, w.HasRunningJobByTaskUUID(taskUUID))

	doneCh := make(chan bool)
	taskChan <- &Task{
		Context: context.Background(),
		UUID:    taskUUID,
		Action:  taskWithDoneCh(doneCh),
	}

	// check when task running
	time.Sleep(time.Microsecond * 100)
	assert.True(t, w.HasRunningJobByTaskUUID(taskUUID))

	// complete task
	doneCh <- true
	<-taskCompletedChan

	// check when task completed
	time.Sleep(time.Microsecond * 100)
	assert.False(t, w.HasRunningJobByTaskUUID(taskUUID))
}

func TestWorker_HoldRunningJobByTaskUUID(t *testing.T) {
	taskChan := make(chan *Task)
	taskCompletedChan := make(chan bool)
	taskUUID := "1"

	w := NewWorker(taskChan, Callbacks{
		TaskStartedCallback: func(_ context.Context, uuid string) error { return nil },
		TaskCompletedCallback: func(_ context.Context, uuid string, log []byte) error {
			taskCompletedChan <- true
			return nil
		},
	})
	go w.Start()

	// check when task not started yet
	assert.False(t, w.HoldRunningJobByTaskUUID(taskUUID, func(job *Job) {}))

	doneCh := make(chan bool)
	taskChan <- &Task{
		Context: context.Background(),
		UUID:    taskUUID,
		Action:  taskWithDoneCh(doneCh),
	}

	// check reading job log
	time.Sleep(time.Microsecond * 100)
	withHold := w.HoldRunningJobByTaskUUID(taskUUID, func(job *Job) {
		expectedLog := []byte("test")

		// emulate log writing in task
		logboek.Context(job.ctx).Log(string(expectedLog))

		assert.Equal(t, expectedLog, job.Log())
	})
	assert.True(t, withHold)

	// complete task
	doneCh <- true
	<-taskCompletedChan

	// check when task completed
	time.Sleep(time.Microsecond * 100)
	assert.False(t, w.HoldRunningJobByTaskUUID(taskUUID, func(job *Job) {}))
}

func infiniteTask(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context canceled")
		}
	}
}

func taskWithDoneCh(doneCh chan bool) func(context.Context) error {
	return func(_ context.Context) error {
		for {
			select {
			case <-doneCh:
				return nil
			}
		}
	}
}
