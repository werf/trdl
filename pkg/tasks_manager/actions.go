package tasks_manager

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/werf/vault-plugin-secrets-trdl/pkg/tasks_manager/worker"
)

var (
	ErrBusy            = errors.New("busy")
	ErrContextCanceled = errors.New("context canceled")
)

const taskReasonInvalidatedTask = "the task canceled due to restart of the plugin"

func (m *Manager) RunTask(ctx context.Context, reqStorage logical.Storage, taskFunc func(context.Context, logical.Storage) error) (string, error) {
	var taskUUID string
	err := m.doTaskWrap(ctx, reqStorage, taskFunc, func(newTaskFunc func(ctx context.Context) error) error {
		busy, err := m.isBusy(ctx, reqStorage)
		if err != nil {
			return err
		}

		if busy {
			return ErrBusy
		}

		taskUUID, err = m.queueTask(ctx, newTaskFunc)
		return err
	})

	return taskUUID, err
}

func (m *Manager) AddOptionalTask(ctx context.Context, reqStorage logical.Storage, taskFunc func(context.Context, logical.Storage) error) (string, bool, error) {
	taskUUID, err := m.RunTask(ctx, reqStorage, taskFunc)
	if err != nil {
		if err == ErrBusy {
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
		taskUUID, err = m.queueTask(ctx, newTaskFunc)

		return err
	})

	return taskUUID, err
}

func (m *Manager) doTaskWrap(ctx context.Context, reqStorage logical.Storage, taskFunc func(context.Context, logical.Storage) error, f func(func(ctx context.Context) error) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// initialize on first task
	if m.Storage == nil {
		m.Storage = reqStorage
		if err := m.invalidateStorage(ctx, reqStorage); err != nil {
			return fmt.Errorf("unable to invalidate storage: %s", err)
		}
	}

	config, err := getConfiguration(ctx, reqStorage)
	if err != nil {
		return fmt.Errorf("unable to get tasks manager configuration: %s", err)
	}

	var taskTimeoutDuration time.Duration
	if config != nil {
		taskTimeoutDuration = config.TaskTimeout
	} else {
		taskTimeoutDuration = defaultTaskTimeoutDuration
	}

	workerTaskFunc := m.WrapTaskFunc(taskFunc, taskTimeoutDuration)

	return f(workerTaskFunc)
}

// WrapTaskFunc separates processing of the context and the taskFunc execution in the background
func (m *Manager) WrapTaskFunc(taskFunc func(context.Context, logical.Storage) error, taskTimeoutDuration time.Duration) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		ctxWithTimeout, ctxCancelFunc := context.WithTimeout(ctx, taskTimeoutDuration)
		defer ctxCancelFunc()

		resCh := make(chan error)
		go func() {
			defer func() {
				p := recover()
				if p == nil || fmt.Sprint(p) == "send on closed channel" {
					return
				}

				panic(p)
			}()

			resCh <- taskFunc(ctxWithTimeout, m.Storage)
		}()

		select {
		case <-ctxWithTimeout.Done():
			close(resCh)
			hclog.L().Debug("task failed: context canceled")
			return ErrContextCanceled
		case err := <-resCh:
			if err != nil {
				hclog.L().Debug(fmt.Sprintf("task failed: %s", err))
				return err
			}

			hclog.L().Debug("task succeeded")
			return nil
		}
	}
}

func (m *Manager) invalidateStorage(ctx context.Context, reqStorage logical.Storage) error {
	var list []string
	for _, state := range []taskState{taskStateRunning, taskStateQueued} {
		prefix := taskStorageKeyPrefix(state)
		l, err := reqStorage.List(ctx, prefix)
		if err != nil {
			return fmt.Errorf("unable to list %q in storage: %s", prefix, err)
		}

		list = append(list, l...)
	}

	for _, uuid := range list {
		if err := switchTaskToCompletedInStorage(ctx, reqStorage, taskStatusCanceled, uuid, switchTaskToCompletedInStorageOptions{
			reason: taskReasonInvalidatedTask,
		}); err != nil {
			return fmt.Errorf("unable to invalidate task %q: %s", uuid, err)
		}
	}

	return nil
}

func (m *Manager) queueTask(ctx context.Context, workerTaskFunc func(context.Context) error) (string, error) {
	queuedTaskUUID, err := addNewTaskToStorage(ctx, m.Storage)
	if err != nil {
		return "", err
	}

	m.taskChan <- &worker.Task{Context: ctx, UUID: queuedTaskUUID, Action: workerTaskFunc}

	return queuedTaskUUID, nil
}

func (m *Manager) isBusy(ctx context.Context, reqStorage logical.Storage) (bool, error) {
	// busy if there are running or queued tasks
	for _, prefix := range []string{storageKeyPrefixRunningTask, storageKeyPrefixQueuedTask} {
		list, err := reqStorage.List(ctx, prefix)
		if err != nil {
			return false, fmt.Errorf("unable to list %q in storage: %s", prefix, err)
		}

		if len(list) != 0 {
			return true, nil
		}
	}

	return false, nil
}
