package queue_manager

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/queue_manager/worker"
)

func (m *Manager) RunTask(ctx context.Context, reqStorage logical.Storage, taskFunc func(context.Context, logical.Storage) error) (string, error) {
	var taskUUID string
	err := m.doTaskWrap(ctx, reqStorage, taskFunc, func(newTaskFunc func(ctx context.Context) error) error {
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
	err := m.doTaskWrap(ctx, reqStorage, taskFunc, func(newTaskFunc func(ctx context.Context) error) error {
		var err error
		taskUUID, err = m.addWorkerTask(ctx, newTaskFunc)

		return err
	})

	return taskUUID, err
}

func (m *Manager) doTaskWrap(ctx context.Context, reqStorage logical.Storage, taskFunc func(context.Context, logical.Storage) error, f func(func(ctx context.Context) error) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.initManager(reqStorage) // initialize on first call

	config, err := getConfiguration(ctx, reqStorage)
	if err != nil {
		return fmt.Errorf("unable to get queue manager configuration: %s", err)
	}

	var taskTimeoutDuration time.Duration
	if config != nil && config.TaskTimeout != "" {
		taskTimeoutDuration, err = time.ParseDuration(config.TaskTimeout)
		if err != nil {
			return fmt.Errorf("unable to parse task timeout duration %q: %s", config.TaskTimeout, err)
		}
	} else {
		taskTimeoutDuration = defaultTaskTimeoutDuration
	}

	workerTaskFunc := func(ctx context.Context) error {
		ctxWithTimeout, ctxCancelFunc := context.WithTimeout(ctx, taskTimeoutDuration)
		defer ctxCancelFunc()

		if err := taskFunc(ctxWithTimeout, m.Storage); err != nil {
			hclog.L().Debug(fmt.Sprintf("task failed: %s", err))
			return err
		}

		hclog.L().Debug(fmt.Sprintf("task succeeded"))
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
