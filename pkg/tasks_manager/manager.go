package tasks_manager

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/tasks_manager/worker"
)

var BusyError = errors.New("busy")

const (
	taskChanSize = 128

	taskStatusQueued    = "QUEUED"
	taskStatusRunning   = "RUNNING"
	taskStatusCompleted = "COMPLETED"
	taskStatusFailed    = "FAILED"
)

type Manager struct {
	Storage logical.Storage
	Worker  worker.Interface

	taskChan chan *worker.Task
	mu       sync.Mutex
}

func NewManager() Interface {
	m := &Manager{taskChan: make(chan *worker.Task, taskChanSize)}

	m.Worker = worker.NewWorker(m.taskChan, worker.Callbacks{
		TaskStartedCallback:   m.taskStartedCallback,
		TaskFailedCallback:    m.taskFailedCallback,
		TaskCompletedCallback: m.taskCompletedCallback,
	})
	go m.Worker.Start()

	return m
}

func (m *Manager) taskStartedCallback(ctx context.Context, uuid string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	err := func() error {
		if err := m.Storage.Delete(ctx, storageKeyCurrentRunningTask); err != nil {
			return fmt.Errorf("unable to delete %q from storage: %q", storageKeyCurrentRunningTask, err)
		}

		task, err := getQueuedTaskFromStorage(ctx, m.Storage, uuid)
		if err != nil {
			return fmt.Errorf("unable to get queued task %q from storage: %q", uuid, err)
		}

		if task == nil {
			return fmt.Errorf("the task %q not found in storage", uuid)
		}

		task.Status = taskStatusRunning
		task.Modified = time.Now()
		if err := putTaskIntoStorage(ctx, m.Storage, task); err != nil {
			return fmt.Errorf("unable to put task %q into the storage: %q", uuid, err)
		}

		if err := m.Storage.Delete(ctx, queuedTaskStorageKey(uuid)); err != nil {
			return fmt.Errorf("unable to delete %q from storage: %q", queuedTaskStorageKey(uuid), err)
		}

		if err := m.Storage.Put(ctx, &logical.StorageEntry{
			Key:   storageKeyCurrentRunningTask,
			Value: []byte(uuid),
		}); err != nil {
			return fmt.Errorf("unable to put %q into the storage: %q", storageKeyCurrentRunningTask, err)
		}

		return nil
	}()

	if err != nil {
		panic("runtime error: " + err.Error())
	}
}

func (m *Manager) taskCompletedCallback(ctx context.Context, uuid string, log []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()

	err := func() error {
		if err := m.Storage.Delete(ctx, storageKeyCurrentRunningTask); err != nil {
			return fmt.Errorf("unable to delete %q from storage: %q", storageKeyCurrentRunningTask, err)
		}

		task, err := getTaskFromStorage(ctx, m.Storage, uuid)
		if err != nil {
			return fmt.Errorf("unable to get task %q from storage: %q", uuid, err)
		}

		if task == nil {
			return fmt.Errorf("the task %q not found in storage", uuid)
		}

		task.Status = taskStatusCompleted
		task.Modified = time.Now()
		if err := putTaskIntoStorage(ctx, m.Storage, task); err != nil {
			return fmt.Errorf("unable to put task %q into the storage: %q", uuid, err)
		}

		if err := m.Storage.Put(ctx, &logical.StorageEntry{
			Key:   taskLogStorageKey(uuid),
			Value: log,
		}); err != nil {
			return fmt.Errorf("unable to put task %q log into the storage: %q", uuid, err)
		}

		return nil
	}()

	if err != nil {
		panic("runtime error: " + err.Error())
	}
}

func (m *Manager) taskFailedCallback(ctx context.Context, uuid string, log []byte, taskErr error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	err := func() error {
		if err := m.Storage.Delete(ctx, storageKeyCurrentRunningTask); err != nil {
			return fmt.Errorf("unable to delete %q from storage: %q", storageKeyCurrentRunningTask, err)
		}

		task, err := getTaskFromStorage(ctx, m.Storage, uuid)
		if err != nil {
			return fmt.Errorf("unable to get task %q from storage: %q", uuid, err)
		}

		if task == nil {
			return fmt.Errorf("the task %q not found in storage", uuid)
		}

		task.Status = taskStatusFailed
		task.Modified = time.Now()
		if taskErr != nil {
			task.Reason = taskErr.Error()
		}
		if err := putTaskIntoStorage(ctx, m.Storage, task); err != nil {
			return fmt.Errorf("unable to put task %q into the storage: %q", uuid, err)
		}

		if err := m.Storage.Put(ctx, &logical.StorageEntry{
			Key:   taskLogStorageKey(uuid),
			Value: log,
		}); err != nil {
			return fmt.Errorf("unable to put task %q log into the storage: %q", uuid, err)
		}

		return nil
	}()

	if err != nil {
		panic("runtime error: " + err.Error())
	}
}
