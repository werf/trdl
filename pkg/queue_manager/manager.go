package queue_manager

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/hashicorp/vault/sdk/logical"

	"github.com/werf/vault-plugin-secrets-trdl/pkg/queue_manager/queue"
)

var QueueBusyError = errors.New("busy")

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

func (m *Manager) RunTask(ctx context.Context, reqStorage logical.Storage, taskFunc func(context.Context, logical.Storage) error) (string, error) {
	var taskUUID string
	var isTaskAdded bool
	err := m.doTaskWrap(reqStorage, taskFunc, func(newTaskFunc func(ctx context.Context) error) error {
		if !m.Queue.IsEmpty() {
			return QueueBusyError
		}

		var err error
		taskUUID, err = m.addQueueTask(ctx, newTaskFunc)
		isTaskAdded = true

		return err
	})

	return taskUUID, err
}

func (m *Manager) AddOptionalTask(ctx context.Context, reqStorage logical.Storage, taskFunc func(context.Context, logical.Storage) error) (string, bool, error) {
	var taskUUID string
	var isTaskAdded bool
	err := m.doTaskWrap(reqStorage, taskFunc, func(newTaskFunc func(ctx context.Context) error) error {
		if !m.Queue.IsEmpty() {
			return nil
		}

		var err error
		taskUUID, err = m.addQueueTask(ctx, newTaskFunc)
		isTaskAdded = true

		return err
	})

	return taskUUID, isTaskAdded, err
}

func (m *Manager) AddTask(ctx context.Context, reqStorage logical.Storage, taskFunc func(context.Context, logical.Storage) error) (string, error) {
	var taskUUID string
	err := m.doTaskWrap(reqStorage, taskFunc, func(newTaskFunc func(ctx context.Context) error) error {
		var err error
		taskUUID, err = m.addQueueTask(ctx, newTaskFunc)

		return err
	})

	return taskUUID, err
}

func (m *Manager) doTaskWrap(reqStorage logical.Storage, taskFunc func(context.Context, logical.Storage) error, f func(func(ctx context.Context) error) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.initManager(reqStorage) // initialize on first call

	queueTaskFunc := func(ctx context.Context) error {
		return taskFunc(ctx, m.Storage)
	}

	return f(queueTaskFunc)
}

func (m *Manager) addQueueTask(ctx context.Context, queueTaskFunc func(context.Context) error) (string, error) {
	task := newTask()
	if err := putTaskIntoStorage(ctx, m.Storage, task); err != nil {
		return "", fmt.Errorf("unable to put task %q into storage: %s", task.UUID, err)
	}

	m.Queue.AddTask(ctx, task.UUID, queueTaskFunc)

	return task.UUID, nil
}
