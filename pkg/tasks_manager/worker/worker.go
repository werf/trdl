package worker

import (
	"context"
	"fmt"
	"sync"
)

type Worker struct {
	currentJob *Job
	taskChan   chan *Task
	stopChan   chan bool
	callbacks  Callbacks

	mu sync.Mutex
}

type Callbacks struct {
	TaskStartedCallback   func(ctx context.Context, uuid string) error
	TaskFailedCallback    func(ctx context.Context, uuid string, log []byte, err error) error
	TaskCompletedCallback func(ctx context.Context, uuid string, log []byte) error
}

func NewWorker(taskChan chan *Task, callbacks Callbacks) Interface {
	return &Worker{callbacks: callbacks, taskChan: taskChan, stopChan: make(chan bool)}
}

func (q *Worker) Start() {
	for {
		select {
		case task := <-q.taskChan:
			func() {
				job := newJob(task)

				q.setCurrentJob(job)
				defer q.resetCurrentJob()

				if callbackErr := q.callbacks.TaskStartedCallback(job.ctx, job.taskUUID); callbackErr != nil {
					panic(fmt.Sprintf("runtime error: %s", callbackErr.Error()))
				}

				if err := job.action(); err != nil {
					if callbackErr := q.callbacks.TaskFailedCallback(job.ctx, job.taskUUID, job.Log(), err); callbackErr != nil {
						panic(fmt.Sprintf("runtime error: %s", callbackErr.Error()))
					}
				} else {
					if callbackErr := q.callbacks.TaskCompletedCallback(job.ctx, job.taskUUID, job.Log()); callbackErr != nil {
						panic(fmt.Sprintf("runtime error: %s", callbackErr.Error()))
					}
				}
			}()
		case <-q.stopChan:
			return
		}

		if q.stopChan == nil {
			return
		}
	}
}

func (q *Worker) HoldRunningJobByTaskUUID(uuid string, do func(job *Job)) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.currentJob == nil || q.currentJob.taskUUID != uuid {
		return false
	}

	do(q.currentJob)

	return true
}

func (q *Worker) HasRunningJobByTaskUUID(uuid string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	return q.currentJob != nil && q.currentJob.taskUUID == uuid
}

func (q *Worker) Stop() {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.currentJob != nil {
		q.currentJob.ctxCancelFunc()
		q.stopChan = nil
	} else {
		close(q.stopChan)
	}
}

func (q *Worker) setCurrentJob(job *Job) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.currentJob = job
}

func (q *Worker) resetCurrentJob() {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.currentJob = nil
}
