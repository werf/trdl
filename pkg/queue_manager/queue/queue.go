package queue

import (
	"context"
	"fmt"
	"sync"
)

type Queue struct {
	currentTask *task
	queueChan   chan *task
	stopChan    chan bool
	callbacks   Callbacks

	mu sync.Mutex
}

type Callbacks struct {
	TaskStartedCallback   func(ctx context.Context, uuid string) error
	TaskFailedCallback    func(ctx context.Context, uuid string, log []byte, err error) error
	TaskCompletedCallback func(ctx context.Context, uuid string, log []byte) error
}

func NewQueue(callbacks Callbacks) *Queue {
	return &Queue{callbacks: callbacks, queueChan: make(chan *task), stopChan: make(chan bool)}
}

func (q *Queue) Start() {
	go func() {
		for {
			select {
			case task := <-q.queueChan:
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
				close(q.queueChan)
				return
			}
		}
	}()
}

func (q *Queue) AddOptionalTask(ctx context.Context, taskFunc func(ctx context.Context) error) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.currentTask != nil {
		return false
	}

	go func() {
		q.queueChan <- newTask(ctx, taskFunc)
	}()

	return true
}

func (q *Queue) AddTask(ctx context.Context, taskFunc func(ctx context.Context) error) {
	go func() {
		q.queueChan <- newTask(ctx, taskFunc)
	}()
}

func (q *Queue) GetTaskLog(uuid string) []byte {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.currentTask == nil || q.currentTask.uuid != uuid {
		return nil
	}

	return q.currentTask.buff.Bytes()
}

func (q *Queue) HasRunningTaskByUUID(uuid string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	return q.currentTask != nil && q.currentTask.uuid != uuid
}

func (q *Queue) Stop() {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.currentTask != nil {
		q.currentTask.ctxCancelFunc()
	}

	go func() {
		q.stopChan <- true
	}()
}

func (q *Queue) setCurrentTask(task *task) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.currentTask = task
}

func (q *Queue) resetCurrentTask() {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.currentTask = nil
}
