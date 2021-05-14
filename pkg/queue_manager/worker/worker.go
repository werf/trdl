package worker

import (
	"context"
	"fmt"
	"sync"
)

type Worker struct {
	currentTask *Task
	taskChan    chan *Task
	stopChan    chan bool
	callbacks   Callbacks

	mu sync.Mutex
}

type Callbacks struct {
	TaskStartedCallback   func(ctx context.Context, uuid string) error
	TaskFailedCallback    func(ctx context.Context, uuid string, log []byte, err error) error
	TaskCompletedCallback func(ctx context.Context, uuid string, log []byte) error
}

func NewWorker(taskChan chan *Task, callbacks Callbacks) *Worker {
	return &Worker{callbacks: callbacks, taskChan: taskChan, stopChan: make(chan bool)}
}

func (q *Worker) Start() {
	for {
		select {
		case task := <-q.taskChan:
			func() {
				q.setCurrentTask(task)
				defer q.resetCurrentTask()

				if callbackErr := q.callbacks.TaskStartedCallback(task.ctx, task.uuid); callbackErr != nil {
					panic(fmt.Sprintf("runtime error: %s", callbackErr.Error()))
				}

				if err := task.action(); err != nil {
					if callbackErr := q.callbacks.TaskFailedCallback(task.ctx, task.uuid, task.buff.Bytes(), err); callbackErr != nil {
						panic(fmt.Sprintf("runtime error: %s", callbackErr.Error()))
					}
				} else {
					if callbackErr := q.callbacks.TaskCompletedCallback(task.ctx, task.uuid, task.buff.Bytes()); callbackErr != nil {
						panic(fmt.Sprintf("runtime error: %s", callbackErr.Error()))
					}
				}
			}()
		case <-q.stopChan:
			close(q.taskChan)
			return
		}
	}
}

func (q *Worker) GetTaskLog(uuid string) []byte {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.currentTask == nil || q.currentTask.uuid != uuid {
		return nil
	}

	return q.currentTask.buff.Bytes()
}

func (q *Worker) IsBusy() bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	return q.currentTask != nil
}

func (q *Worker) HasRunningTaskByUUID(uuid string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	return q.currentTask != nil && q.currentTask.uuid != uuid
}

func (q *Worker) Stop() {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.currentTask != nil {
		q.currentTask.ctxCancelFunc()
	}

	q.stopChan <- true
}

func (q *Worker) setCurrentTask(task *Task) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.currentTask = task
}

func (q *Worker) resetCurrentTask() {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.currentTask = nil
}
