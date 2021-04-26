package tasks

import (
	"context"
	"fmt"
	"sync"
	"time"

	uuid "github.com/satori/go.uuid"
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
	// TODO RunnerLog bytes.Buffer + autocreate logboek context
	CreatedAt time.Time
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

func (q *Queue) RunTask(runner func(ctx context.Context) error) string {
	task := &Task{
		ID:        uuid.NewV4().String(),
		Runner:    runner,
		CreatedAt: time.Now(),
		Status:    TaskPending,
	}

	q.mu.Lock()
	defer q.mu.Unlock()

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

func (q *Queue) GetTaskStatus(id string) (taskStatus TaskStatus, taskErr error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	task, hasTask := q.Tasks[id]
	if !hasTask {
		return "", fmt.Errorf("no such task %q", id)
	}

	return task.Status, task.RunnerErr
}

func (q *Queue) runNextTask() {
	var nextTask *Task

	func() {
		q.mu.Lock()
		defer q.mu.Unlock()

		if len(q.Tasks) == 0 {
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

	if err := nextTask.Runner(q.RunnerCtx); err != nil {
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
