package tasks_manager

import (
	"context"
	"time"

	"github.com/hashicorp/vault/sdk/logical"
)

const (
	storageKeyCurrentRunningTask = "current_running_task"
	storageKeyPrefixQueuedTask   = "queued_task-"
	storageKeyPrefixTask         = "task-"
	storageKeyPrefixTaskLog      = "task_log-"

	staleTaskReason = "the unfinished task from the previous run"
)

func markStaleTaskAsFailed(ctx context.Context, storage logical.Storage) error {
	uuid, err := getCurrentTaskUUIDFromStorage(ctx, storage)
	if err != nil {
		return err
	}

	if uuid == "" {
		return nil
	}

	task, err := getTaskFromStorage(ctx, storage, uuid)
	if err != nil {
		return err
	}

	if task == nil {
		return nil
	}

	task.Status = taskStatusFailed
	task.Modified = time.Now()
	task.Reason = staleTaskReason
	if err := putTaskIntoStorage(ctx, storage, task); err != nil {
		return err
	}

	return nil
}

func getCurrentTaskUUIDFromStorage(ctx context.Context, storage logical.Storage) (string, error) {
	currentRunningTaskValue, err := storage.Get(ctx, storageKeyCurrentRunningTask)
	if err != nil {
		return "", err
	}

	if currentRunningTaskValue == nil {
		return "", nil
	}

	return string(currentRunningTaskValue.Value), nil
}

func getQueuedTaskFromStorage(ctx context.Context, storage logical.Storage, uuid string) (*Task, error) {
	return getTaskFromStorageBase(ctx, storage, queuedTaskStorageKey(uuid))
}

func getTaskFromStorage(ctx context.Context, storage logical.Storage, uuid string) (*Task, error) {
	return getTaskFromStorageBase(ctx, storage, taskStorageKey(uuid))
}

func getTaskFromStorageBase(ctx context.Context, storage logical.Storage, taskStorageKey string) (*Task, error) {
	e, err := storage.Get(ctx, taskStorageKey)
	if err != nil {
		return nil, err
	}

	if e == nil {
		return nil, nil
	}

	return storageEntryToTask(e)
}

func getTaskLogFromStorage(ctx context.Context, storage logical.Storage, uuid string) ([]byte, error) {
	e, err := storage.Get(ctx, taskLogStorageKey(uuid))
	if err != nil {
		return nil, err
	}

	if e == nil {
		return nil, nil
	}

	return e.Value, nil
}

func putQueuedTaskIntoStorage(ctx context.Context, storage logical.Storage, task *Task) error {
	e, err := queuedTaskToStorageEntry(task)
	if err != nil {
		return err
	}

	return storage.Put(ctx, e)
}

func putTaskIntoStorage(ctx context.Context, storage logical.Storage, task *Task) error {
	e, err := taskToStorageEntry(task)
	if err != nil {
		return err
	}

	return storage.Put(ctx, e)
}
