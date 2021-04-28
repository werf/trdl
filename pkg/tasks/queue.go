package tasks

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/werf/logboek"
)

type TaskStatus string

const (
	TaskPending   = "TaskPending"
	TaskRunning   = "TaskRunning"
	TaskSucceeded = "TaskSucceeded"
	TaskFailed    = "TaskFailed"
)

type Task struct {
	ID        string
	Status    TaskStatus
	RunnerErr error
	Runner    func(ctx context.Context) error
	CreatedAt time.Time

	LogBuffer *bytes.Buffer
}

func NewTask(runner func(ctx context.Context) error) *Task {
	return &Task{
		ID:        uuid.NewV4().String(),
		Runner:    runner,
		CreatedAt: time.Now(),
		Status:    TaskPending,
		LogBuffer: bytes.NewBuffer([]byte{}),
	}
}

type Queue struct {
	RunnerCtx  context.Context
	TasksQueue []string
	Tasks      map[string]*Task

	mu sync.Mutex
}

func NewQueue(runnerCtx context.Context) *Queue {
	return &Queue{
		RunnerCtx: runnerCtx,
		Tasks:     make(map[string]*Task),
	}
}

// RunScheduledTask used to add a new task to the queue only if there are no tasks already queued or running
func (q *Queue) RunScheduledTask(runner func(ctx context.Context) error) string {
	q.mu.Lock()
	defer q.mu.Unlock()

	for _, task := range q.Tasks {
		switch task.Status {
		case TaskRunning, TaskPending:
			return ""
		}
	}

	task := NewTask(runner)

	q.Tasks[task.ID] = task
	q.TasksQueue = append(q.TasksQueue, task.ID)

	go q.runNextTask()

	return task.ID
}

// RunQueuedTask used to add a new task to the queue
func (q *Queue) RunQueuedTask(runner func(ctx context.Context) error) string {
	q.mu.Lock()
	defer q.mu.Unlock()

	task := NewTask(runner)

	q.Tasks[task.ID] = task
	q.TasksQueue = append(q.TasksQueue, task.ID)

	go q.runNextTask()

	return task.ID
}

func (q *Queue) HasTask(id string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	_, hasTask := q.Tasks[id]
	return hasTask
}

func (q *Queue) GetTaskData(id string) (taskStatus TaskStatus, logs []byte, taskErr error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	task, hasTask := q.Tasks[id]
	if !hasTask {
		return "", nil, fmt.Errorf("no such task %q", id)
	}

	return task.Status, task.LogBuffer.Bytes(), task.RunnerErr
}

func (q *Queue) runNextTask() {
	var nextTask *Task

	func() {
		q.mu.Lock()
		defer q.mu.Unlock()

		if len(q.TasksQueue) == 0 {
			return
		}

		id := q.TasksQueue[0]
		q.TasksQueue = q.TasksQueue[1:]

		task, hasTask := q.Tasks[id]
		if !hasTask {
			panic("unexpected condition")
		}

		nextTask = task
		task.Status = TaskRunning
	}()

	if nextTask == nil {
		return
	}

	loggerCtx := logboek.NewContext(q.RunnerCtx, logboek.DefaultLogger().NewSubLogger(nextTask.LogBuffer, nextTask.LogBuffer))

	if err := nextTask.Runner(loggerCtx); err != nil {
		func() {
			q.mu.Lock()
			defer q.mu.Unlock()

			nextTask.Status = TaskFailed
			nextTask.RunnerErr = err
		}()
	} else {
		func() {
			q.mu.Lock()
			defer q.mu.Unlock()

			nextTask.Status = TaskSucceeded
			nextTask.RunnerErr = err
		}()
	}

	go q.runNextTask()
}
