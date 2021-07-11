package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/werf/logboek"
)

const (
	TaskFailedCallback    = "FAILED"
	TaskCompletedCallback = "COMPLETED"
)

var infiniteTaskContextCanceledError = errors.New(contextCanceledError.Error() + " (infinite)")

func TestTaskCallbacks(t *testing.T) {
	taskChan := make(chan *Task)
	taskProcessedChan := make(chan bool)

	var expectedUUID string
	var expectedCallback string
	var expectedErr error
	var expectedLog []byte

	w := NewWorker(taskChan, Callbacks{
		TaskStartedCallback: func(_ context.Context, uuid string) {
			assert.Equal(t, expectedUUID, uuid)
		},
		TaskFailedCallback: func(_ context.Context, uuid string, log []byte, err error) {
			assert.Equal(t, expectedCallback, TaskFailedCallback)
			assert.Equal(t, expectedUUID, uuid)
			assert.Equal(t, expectedErr, err)
			assert.Equal(t, expectedLog, log)
			taskProcessedChan <- true
		},
		TaskCompletedCallback: func(_ context.Context, uuid string, log []byte) {
			assert.Equal(t, expectedCallback, TaskCompletedCallback)
			assert.Equal(t, uuid, expectedUUID)
			assert.Nil(t, expectedErr)
			assert.Equal(t, log, expectedLog)
			taskProcessedChan <- true
		},
	})
	go w.Start()

	for _, c := range []struct {
		testName string
		uuid     string
		callback string
		log      []byte
		err      error
	}{
		{
			testName: "completed",
			uuid:     "1",
			callback: TaskCompletedCallback,
			log:      []byte("hello"),
			err:      nil,
		},
		{
			testName: "failed",
			uuid:     "2",
			callback: TaskFailedCallback,
			log:      []byte("error"),
			err:      errors.New("error"),
		},
	} {
		t.Run(c.testName, func(t *testing.T) {
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
		})
	}
}

func TestWorker_Stop(t *testing.T) {
	taskChan := make(chan *Task)
	workerStoppedChan := make(chan bool)

	w := NewWorker(taskChan, Callbacks{})
	go func() {
		w.Start()

		workerStoppedChan <- true
	}()

	go w.Stop()

	// check worker stopped
	<-workerStoppedChan
}

func TestWorker_StopWithRunningJob(t *testing.T) {
	taskChan := make(chan *Task)
	taskFailedChan := make(chan bool)
	workerStoppedChan := make(chan bool)
	taskUUID := "1"
	queuedTaskUUID := "2"

	w := NewWorker(taskChan, Callbacks{
		TaskStartedCallback: func(_ context.Context, uuid string) {},
		TaskFailedCallback: func(_ context.Context, uuid string, _ []byte, _ error) {
			assert.Equal(t, taskUUID, uuid)
			taskFailedChan <- true
		},
	})

	// start processing tasks
	go func() {
		w.Start()
		workerStoppedChan <- true
	}()

	// add task
	taskChan <- stopTask(taskUUID)

	// queue another task
	go func() {
		taskChan <- stopTask(queuedTaskUUID)
	}()

	// give time for the task to become active
	time.Sleep(time.Microsecond * 100)

	// cancel running task and stop worker
	go w.Stop()

	// check task failed
	<-taskFailedChan

	// check worker stopped
	<-workerStoppedChan

	// check queued task
	<-taskChan
}

func TestWorker_HasRunningJobByTaskUUID(t *testing.T) {
	taskChan := make(chan *Task)
	taskCompletedChan := make(chan bool)
	taskUUID := "1"

	w := NewWorker(taskChan, Callbacks{
		TaskStartedCallback: func(_ context.Context, uuid string) {},
		TaskCompletedCallback: func(ctx context.Context, uuid string, log []byte) {
			taskCompletedChan <- true
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

	// give time for the task to become active
	time.Sleep(time.Microsecond * 100)

	// check when task running
	assert.True(t, w.HasRunningJobByTaskUUID(taskUUID))

	// complete task
	doneCh <- true
	<-taskCompletedChan

	// give time to reset current task
	time.Sleep(time.Microsecond * 100)

	// check when task completed
	assert.False(t, w.HasRunningJobByTaskUUID(taskUUID))
}

func TestWorker_HoldRunningJobByTaskUUID(t *testing.T) {
	taskChan := make(chan *Task)
	taskCompletedChan := make(chan bool)
	taskUUID := "1"

	w := NewWorker(taskChan, Callbacks{
		TaskStartedCallback: func(_ context.Context, uuid string) {},
		TaskCompletedCallback: func(_ context.Context, uuid string, log []byte) {
			taskCompletedChan <- true
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

	// give time for the task to become active
	time.Sleep(time.Microsecond * 100)

	// check reading job log
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

	// give time to reset current task
	time.Sleep(time.Microsecond * 100)

	// check when task completed
	assert.False(t, w.HoldRunningJobByTaskUUID(taskUUID, func(job *Job) {}))
}

func infiniteTask(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return infiniteTaskContextCanceledError
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

func stopTask(uuid string) *Task {
	return &Task{
		Context: context.Background(),
		UUID:    uuid,
		Action:  infiniteTask,
	}
}
