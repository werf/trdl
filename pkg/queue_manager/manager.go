package queue_manager

import (
	"errors"
	"sync"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/queue_manager/worker"
)

var QueueBusyError = errors.New("busy")

const (
	numberOfWorkers = 1
)

type Manager struct {
	Storage logical.Storage
	Workers []*worker.Worker

	taskChan chan *worker.Task
	mu       sync.Mutex
}

func NewManager() Interface {
	return &Manager{taskChan: make(chan *worker.Task)}
}

func (m *Manager) initManager(storage logical.Storage) {
	if len(m.Workers) < numberOfWorkers {
		for i := 0; i < numberOfWorkers-len(m.Workers); i++ {
			m.startWorker()
		}
	}

	if m.Storage != nil {
		return
	}

	m.Storage = storage
}
