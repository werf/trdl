package worker

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/werf/logboek"
)

const (
	TaskStartedCallback   = "TaskStartedCallback"
	TaskFailedCallback    = "TaskFailedCallback"
	TaskSucceededCallback = "TaskSucceededCallback"
)

var errTestTaskContextCanceled = errors.New("no offense, but it's over: context canceled")

type MockedTasksCallbacks struct {
	mock.Mock
}

func (m *MockedTasksCallbacks) TaskStartedCallback(_ context.Context, uuid string) {
	m.Called(uuid)
}

func (m *MockedTasksCallbacks) TaskFailedCallback(_ context.Context, uuid string, log []byte, err error) {
	m.Called(uuid, log, err)
}

func (m *MockedTasksCallbacks) TaskSucceededCallback(_ context.Context, uuid string, log []byte) {
	m.Called(uuid, log)
}

func TestWorkerContext(t *testing.T) {
	ctx := context.Background()
	workerCtx, workerCtxCancelFunc := context.WithCancel(ctx)
	taskChan := make(chan *Task)
	workerFinishedChan := make(chan bool)

	w := NewWorker(workerCtx, taskChan, nil)

	// start processing tasks
	go func() {
		w.Start()
		workerFinishedChan <- true
	}()

	// wait till the worker is finished
	workerCtxCancelFunc()
	<-workerFinishedChan
}

func TestTaskCallbacks(t *testing.T) {
	for _, c := range []struct {
		testName         string
		taskUUID         string
		taskLog          []byte
		taskErr          error
		expectedCallback string
	}{
		{
			testName:         TaskSucceededCallback,
			taskUUID:         "1",
			taskLog:          []byte("hello"),
			expectedCallback: TaskSucceededCallback,
		},
		{
			testName:         TaskFailedCallback,
			taskUUID:         "2",
			taskLog:          []byte("error"),
			taskErr:          errors.New("error"),
			expectedCallback: TaskFailedCallback,
		},
	} {
		t.Run(c.testName, func(t *testing.T) {
			taskChan := make(chan *Task)
			mockedTasksCallbacks := &MockedTasksCallbacks{}
			ctx := context.Background()
			workerCtx, workerCtxCancelFunc := context.WithCancel(ctx)
			workerFinishedChan := make(chan bool)
			w := NewWorker(workerCtx, taskChan, mockedTasksCallbacks)

			// start processing tasks
			go func() {
				w.Start()
				workerFinishedChan <- true
			}()

			// setup callback expectations
			mockedTasksCallbacks.On(TaskStartedCallback, c.taskUUID).Return()
			switch c.expectedCallback {
			case TaskFailedCallback:
				mockedTasksCallbacks.On(TaskFailedCallback, c.taskUUID, c.taskLog, c.taskErr).Return()
			case TaskSucceededCallback:
				mockedTasksCallbacks.On(TaskSucceededCallback, c.taskUUID, c.taskLog).Return()
			}

			doneCh := make(chan bool)
			taskChan <- &Task{
				Context: context.Background(),
				UUID:    c.taskUUID,
				Action: func(ctx context.Context) error {
					defer func() { doneCh <- true }()

					logboek.Context(ctx).Log(string(c.taskLog))

					if c.taskErr != nil {
						return c.taskErr
					}

					return nil
				},
			}

			// wait till the task is completed
			<-doneCh
			workerCtxCancelFunc()
			<-workerFinishedChan

			mockedTasksCallbacks.AssertExpectations(t)
		})
	}
}

func TestWorker_CancelRunningJobByTaskUUID(t *testing.T) {
	taskChan := make(chan *Task, 2)
	taskUUID := "1"
	queuedTaskUUID := "2"

	mockedTasksCallbacks := &MockedTasksCallbacks{}
	w := NewWorker(context.Background(), taskChan, mockedTasksCallbacks)

	// check nothing to cancel
	canceled := w.CancelRunningJobByTaskUUID(taskUUID)
	assert.False(t, canceled)

	// queue task
	task1Channels, task1 := testTask(taskUUID)
	taskChan <- task1

	// queue another task
	task2Channels, task2 := testTask(queuedTaskUUID)
	taskChan <- task2

	// setup callback expectations
	mockedTasksCallbacks.On(TaskStartedCallback, taskUUID).Return()
	mockedTasksCallbacks.On(TaskStartedCallback, queuedTaskUUID).Return()
	mockedTasksCallbacks.On(TaskFailedCallback, taskUUID, []byte{}, errTestTaskContextCanceled).Return()

	// start processing tasks
	go w.Start()

	// wait till the task is started
	<-task1Channels.startedCh

	// cancel running task
	canceled = w.CancelRunningJobByTaskUUID(taskUUID)
	assert.True(t, canceled)

	// wait till the task is completed with context canceled error
	<-task1Channels.completedCh

	// check the next task started
	<-task2Channels.startedCh

	mockedTasksCallbacks.AssertExpectations(t)
}

func TestWorker_HoldRunningJobByTaskUUID(t *testing.T) {
	ctx := context.Background()
	workerCtx, workerCtxCancelFunc := context.WithCancel(ctx)
	workerFinishedChan := make(chan bool)
	taskChan := make(chan *Task, 1)
	taskUUID := "1"
	expectedLog := []byte("test")

	mockedTasksCallbacks := &MockedTasksCallbacks{}
	w := NewWorker(workerCtx, taskChan, mockedTasksCallbacks)

	// try to hold when task not started yet
	assert.False(t, w.HoldRunningJobByTaskUUID(taskUUID, func(job *Job) {}))

	// queue task
	taskChannels, task := testTask(taskUUID)
	taskChan <- task

	// setup callback expectations
	mockedTasksCallbacks.On(TaskStartedCallback, taskUUID).Return()
	mockedTasksCallbacks.On(TaskSucceededCallback, taskUUID, expectedLog).Return()

	// start processing tasks
	go func() {
		w.Start()
		workerFinishedChan <- true
	}()

	// wait till the task is started
	<-taskChannels.startedCh

	// check reading job log
	withHold := w.HoldRunningJobByTaskUUID(taskUUID, func(job *Job) {
		// send log message
		taskChannels.msgCh <- string(expectedLog)
		<-taskChannels.msgSentCh

		// complete task
		taskChannels.doneCh <- true
		<-taskChannels.completedCh

		assert.Equal(t, expectedLog, job.Log())
	})
	assert.True(t, withHold)

	// wait till the task is completed and finish worker
	workerCtxCancelFunc()
	<-workerFinishedChan

	mockedTasksCallbacks.AssertExpectations(t)
}

type testTaskChannels struct {
	startedCh   chan bool
	msgCh       chan string
	msgSentCh   chan bool
	doneCh      chan bool
	completedCh chan bool
}

func testTask(uuid string) (testTaskChannels, *Task) {
	channels := testTaskChannels{
		startedCh:   make(chan bool),
		msgCh:       make(chan string),
		msgSentCh:   make(chan bool),
		doneCh:      make(chan bool),
		completedCh: make(chan bool),
	}

	task := &Task{
		Context: context.Background(),
		UUID:    uuid,
		Action:  testTaskAction(channels),
	}

	return channels, task
}

func testTaskAction(channels testTaskChannels) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		defer func() { channels.completedCh <- true }()

		channels.startedCh <- true

		for {
			select {
			case log := <-channels.msgCh:
				logboek.Context(ctx).Log(log)
				channels.msgSentCh <- true
			case <-channels.doneCh:
				return nil
			case <-ctx.Done():
				return errTestTaskContextCanceled
			}
		}
	}
}
