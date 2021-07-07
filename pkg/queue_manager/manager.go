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
	m := &Manager{taskChan: make(chan *worker.Task)}

	for i := 0; i < numberOfWorkers; i++ {
		m.startWorker()
	}

	return m
}
