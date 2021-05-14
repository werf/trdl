package queue_manager

import (
	"context"
	"fmt"
	"os"
	"syscall"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/queue_manager/worker"
)

func (m *Manager) RunTask(ctx context.Context, reqStorage logical.Storage, taskFunc func(context.Context, logical.Storage) error) (string, error) {
	var taskUUID string
	err := m.doTaskWrap(reqStorage, taskFunc, func(newTaskFunc func(ctx context.Context) error) error {
		allWorkersBusy := true
		for _, w := range m.Workers {
			if !w.IsBusy() {
				allWorkersBusy = false
				break
			}
		}

		if allWorkersBusy {
			return QueueBusyError
		}

		var err error
		taskUUID, err = m.addWorkerTask(ctx, newTaskFunc)
		return err
	})

	return taskUUID, err
}

func (m *Manager) AddOptionalTask(ctx context.Context, reqStorage logical.Storage, taskFunc func(context.Context, logical.Storage) error) (string, bool, error) {
	taskUUID, err := m.RunTask(ctx, reqStorage, taskFunc)
	if err != nil {
		if err == QueueBusyError {
			return taskUUID, false, nil
		}

		return "", false, err
	}

	return taskUUID, true, nil
}

func (m *Manager) AddTask(ctx context.Context, reqStorage logical.Storage, taskFunc func(context.Context, logical.Storage) error) (string, error) {
	var taskUUID string
	err := m.doTaskWrap(reqStorage, taskFunc, func(newTaskFunc func(ctx context.Context) error) error {
		var err error
		taskUUID, err = m.addWorkerTask(ctx, newTaskFunc)

		return err
	})

	return taskUUID, err
}

func (m *Manager) doTaskWrap(reqStorage logical.Storage, taskFunc func(context.Context, logical.Storage) error, f func(func(ctx context.Context) error) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.initManager(reqStorage) // initialize on first call

	workerTaskFunc := func(ctx context.Context) error {
		stderr := os.NewFile(uintptr(syscall.Stderr), "/dev/stderr")

		if err := taskFunc(ctx, m.Storage); err != nil {
			fmt.Fprintf(stderr, "task failed: %s\n", err) // Remove this debug when tasks log debugged
			return err
		}

		fmt.Fprintf(stderr, "task succeeded\n") // Remove this debug when tasks log debugged
		return nil
	}

	return f(workerTaskFunc)
}

func (m *Manager) addWorkerTask(ctx context.Context, workerTaskFunc func(context.Context) error) (string, error) {
	task := newTask()
	if err := putTaskIntoStorage(ctx, m.Storage, task); err != nil {
		return "", fmt.Errorf("unable to put task %q into storage: %s", task.UUID, err)
	}

	go func() { m.taskChan <- worker.NewTask(ctx, task.UUID, workerTaskFunc) }()

	return task.UUID, nil
}
