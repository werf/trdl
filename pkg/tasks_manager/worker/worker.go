package worker

import (
	"context"
	"sync"
)

type Worker struct {
	ctx        context.Context
	currentJob *Job
	taskChan   chan *Task
	callbacks  Callbacks

	mu sync.Mutex
}

type Callbacks struct {
	TaskStartedCallback   func(ctx context.Context, uuid string)
	TaskFailedCallback    func(ctx context.Context, uuid string, log []byte, err error)
	TaskSucceededCallback func(ctx context.Context, uuid string, log []byte)
}

func NewWorker(ctx context.Context, taskChan chan *Task, callbacks Callbacks) Interface {
	return &Worker{ctx: ctx, callbacks: callbacks, taskChan: taskChan}
}

func (w *Worker) Start() {
	for {
		select {
		case task := <-w.taskChan:
			func() {
				job := newJob(task)
				w.setCurrentJob(job)
				defer w.resetCurrentJob()

				w.callbacks.TaskStartedCallback(w.ctx, job.taskUUID)
				if err := job.action(); err != nil {
					w.callbacks.TaskFailedCallback(w.ctx, job.taskUUID, job.Log(), err)
				} else {
					w.callbacks.TaskSucceededCallback(w.ctx, job.taskUUID, job.Log())
				}
			}()
		case <-w.ctx.Done():
			return
		}
	}
}

func (w *Worker) HoldRunningJobByTaskUUID(uuid string, do func(job *Job)) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.currentJob == nil || w.currentJob.taskUUID != uuid {
		return false
	}

	do(w.currentJob)

	return true
}

func (w *Worker) CancelRunningJobByTaskUUID(uuid string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.currentJob != nil && w.currentJob.taskUUID == uuid {
		w.currentJob.ctxCancelFunc()
		return true
	}

	return false
}

func (w *Worker) setCurrentJob(job *Job) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.currentJob = job
}

func (w *Worker) resetCurrentJob() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.currentJob = nil
}
