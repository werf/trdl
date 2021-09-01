package tasks_manager

import (
	"context"
	"sync"

	"github.com/hashicorp/vault/sdk/logical"

	"github.com/werf/trdl/server/pkg/tasks_manager/worker"
)

const taskChanSize = 128

type Manager struct {
	Storage logical.Storage
	Worker  worker.Interface

	taskChan chan *worker.Task
	mu       sync.Mutex
}

func NewManager() *Manager {
	m := &Manager{taskChan: make(chan *worker.Task, taskChanSize)}
	m.Worker = worker.NewWorker(context.Background(), m.taskChan, m)
	go m.Worker.Start()

	return m
}

func (m *Manager) TaskStartedCallback(ctx context.Context, uuid string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := switchTaskToRunningInStorage(ctx, m.Storage, uuid); err != nil {
		panic("runtime error: " + err.Error())
	}
}

func (m *Manager) TaskSucceededCallback(ctx context.Context, uuid string, log []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := switchTaskToCompletedInStorage(ctx, m.Storage, taskStatusSucceeded, uuid, switchTaskToCompletedInStorageOptions{
		log: log,
	}); err != nil {
		panic("runtime error: " + err.Error())
	}
}

func (m *Manager) TaskFailedCallback(ctx context.Context, uuid string, log []byte, taskErr error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := switchTaskToCompletedInStorage(ctx, m.Storage, taskStatusFailed, uuid, switchTaskToCompletedInStorageOptions{
		reason: taskErr.Error(),
		log:    log,
	}); err != nil {
		panic("runtime error: " + err.Error())
	}
}
