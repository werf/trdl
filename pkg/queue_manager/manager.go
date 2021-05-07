package queue_manager

import (
	"context"
	"sync"

	"github.com/hashicorp/vault/sdk/logical"

	"github.com/werf/vault-plugin-secrets-trdl/pkg/queue_manager/queue"
)

type Manager struct {
	Storage logical.Storage
	Queue   *queue.Queue

	mu sync.Mutex
}

func NewManager() Interface {
	return &Manager{}
}

func (m *Manager) initManager(storage logical.Storage) {
	if m.Storage != nil {
		return
	}

	m.Storage = storage
	m.initQueue()
}

func (m *Manager) RunOptionalTask(ctx context.Context, reqStorage logical.Storage, taskFunc func(ctx context.Context, storage logical.Storage) error) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.initManager(reqStorage) // initialize on first call

	queueTaskFunc := func(ctx context.Context) error {
		return taskFunc(ctx, m.Storage)
	}

	return m.Queue.AddOptionalTask(ctx, queueTaskFunc)
}

func (m *Manager) RunTask(ctx context.Context, reqStorage logical.Storage, taskFunc func(ctx context.Context, storage logical.Storage) error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.initManager(reqStorage) // initialize on first call

	queueTaskFunc := func(ctx context.Context) error {
		return taskFunc(ctx, m.Storage)
	}

	m.Queue.AddTask(ctx, queueTaskFunc)
}
