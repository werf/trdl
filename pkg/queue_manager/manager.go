package queue_manager

import (
	"errors"
	"sync"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/queue_manager/worker"
)

var QueueBusyError = errors.New("busy")

type Manager struct {
	Storage logical.Storage
	Worker  *worker.Worker

	taskChan chan *worker.Task
	mu       sync.Mutex
}

func NewManager() Interface {
	return &Manager{taskChan: make(chan *worker.Task)}
}

func (m *Manager) initManager(storage logical.Storage) {
	if m.Storage != nil {
		return
	}

	m.Storage = storage
	m.startWorker()
}
