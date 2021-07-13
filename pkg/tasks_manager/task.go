package tasks_manager

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/vault/sdk/logical"
	uuid "github.com/satori/go.uuid"
)

type (
	taskStatus string
	taskState  string
)

const (
	taskStatusQueued    taskStatus = "QUEUED"
	taskStatusRunning   taskStatus = "RUNNING"
	taskStatusSucceeded taskStatus = "SUCCEEDED"
	taskStatusFailed    taskStatus = "FAILED"
	taskStatusCanceled  taskStatus = "CANCELED"

	taskStateQueued    taskState = "QUEUED"
	taskStateRunning   taskState = "RUNNING"
	taskStateCompleted taskState = "COMPLETED"

	storageKeyPrefixQueuedTask    = "queued_task/"
	storageKeyPrefixRunningTask   = "running_task/"
	storageKeyPrefixCompletedTask = "completed_task/"
	storageKeyPrefixTaskLog       = "task_log/"
)

var taskStateStatusesCompleted = []taskStatus{taskStatusSucceeded, taskStatusFailed, taskStatusCanceled}

type Task struct {
	UUID     string    `structs:"uuid" json:"uuid"`
	Status   string    `structs:"status" json:"status"`
	Reason   string    `structs:"reason" json:"reason"`
	Created  time.Time `structs:"created" json:"created"`
	Modified time.Time `structs:"modified" json:"modified"`
}

func newTask() *Task {
	task := &Task{}
	task.UUID = uuid.NewV4().String()
	task.Status = string(taskStatusQueued)

	tNow := time.Now()
	task.Created = tNow
	task.Modified = tNow

	return task
}

func addNewTaskToStorage(ctx context.Context, storage logical.Storage) (string, error) {
	queuedTask := newTask()
	storageKey := taskStorageKey(taskStateQueued, queuedTask.UUID)
	entry, err := logical.StorageEntryJSON(storageKey, queuedTask)
	if err != nil {
		return "", fmt.Errorf("unable to prepare storage entry JSON: %s", err)
	}

	if err := storage.Put(ctx, entry); err != nil {
		return "", fmt.Errorf("unable to put %q into storage: %s", storageKey, err)
	}

	return queuedTask.UUID, nil
}

func switchTaskToRunningInStorage(ctx context.Context, storage logical.Storage, uuid string) error {
	// get previous task state from storage
	var prevTask *Task
	{
		t, err := getTaskFromStorage(ctx, storage, taskStateQueued, uuid)
		if err != nil {
			return err
		}

		if t == nil {
			return fmt.Errorf("queued task %q must be in storage", uuid)
		}

		prevTask = t
	}

	// add running task to storage
	{
		runningTask := prevTask
		runningTask.Status = string(taskStatusRunning)
		runningTask.Modified = time.Now()
		runningTaskState := taskStateRunning

		storageKey := taskStorageKey(runningTaskState, uuid)
		entry, err := logical.StorageEntryJSON(storageKey, prevTask)
		if err != nil {
			return fmt.Errorf("unable to prepare storage entry JSON: %s", err)
		}

		if err := storage.Put(ctx, entry); err != nil {
			return fmt.Errorf("unable to put %q into storage: %s", storageKey, err)
		}
	}

	// delete previous state from storage
	prevStorageKey := taskStorageKey(taskStateQueued, uuid)
	if err := storage.Delete(ctx, prevStorageKey); err != nil {
		return fmt.Errorf("unable to delete %q from storage: %q", prevStorageKey, err)
	}

	return nil
}

type switchTaskToCompletedInStorageOptions struct {
	reason string
	log    []byte
}

func switchTaskToCompletedInStorage(ctx context.Context, storage logical.Storage, status taskStatus, uuid string, opts switchTaskToCompletedInStorageOptions) error {
	// validate new status
	if !isCompletedTaskStatus(status) {
		return fmt.Errorf("runtime error: task in completed state cannot be with status %q", status)
	}

	// get previous task state from storage
	var prevTask *Task
	var prevTaskState taskState
	{
		for _, s := range []taskState{taskStateRunning, taskStateQueued} {
			t, err := getTaskFromStorage(ctx, storage, s, uuid)
			if err != nil {
				return err
			}

			if t == nil {
				continue
			}

			prevTask = t
			prevTaskState = s
		}

		if prevTask == nil {
			return fmt.Errorf("queued or running task %q not found in storage", uuid)
		}
	}

	// add completed task and optional log to storage
	{
		completedTask := prevTask
		completedTask.Status = string(status)
		completedTask.Modified = time.Now()
		completedTask.Reason = opts.reason
		completedTaskState := taskStatusState(status)

		storageKey := taskStorageKey(completedTaskState, uuid)
		entry, err := logical.StorageEntryJSON(storageKey, completedTask)
		if err != nil {
			return fmt.Errorf("unable to prepare storage entry JSON: %s", err)
		}

		if err := storage.Put(ctx, entry); err != nil {
			return fmt.Errorf("unable to put %q into storage: %s", storageKey, err)
		}

		if len(opts.log) != 0 {
			logStorageKey := taskLogStorageKey(uuid)
			if err := storage.Put(ctx, &logical.StorageEntry{
				Key:   logStorageKey,
				Value: opts.log,
			}); err != nil {
				return fmt.Errorf("unable to put %q into the storage: %q", logStorageKey, err)
			}
		}
	}

	// delete previous state from storage
	prevStorageKey := taskStorageKey(prevTaskState, prevTask.UUID)
	if err := storage.Delete(ctx, prevStorageKey); err != nil {
		return fmt.Errorf("unable to delete %q from storage: %q", prevStorageKey, err)
	}

	return nil
}

func getTaskFromStorage(ctx context.Context, storage logical.Storage, state taskState, uuid string) (*Task, error) {
	storageKey := taskStorageKey(state, uuid)
	entry, err := storage.Get(ctx, storageKey)
	if err != nil {
		return nil, fmt.Errorf("unable to get %q from storage: %s", storageKey, err)
	}

	if entry == nil {
		return nil, nil
	}

	return storageEntryToTask(entry)
}

func getTaskLogFromStorage(ctx context.Context, storage logical.Storage, uuid string) ([]byte, error) {
	storageKey := taskLogStorageKey(uuid)
	entry, err := storage.Get(ctx, taskLogStorageKey(uuid))
	if err != nil {
		return nil, fmt.Errorf("unable to get %q from storage: %s", storageKey, err)
	}

	if entry == nil {
		return nil, nil
	}

	return entry.Value, nil
}

func taskStorageKey(state taskState, uuid string) string {
	return taskStorageKeyPrefix(state) + uuid
}

func taskStorageKeyPrefix(state taskState) string {
	switch state {
	case taskStateQueued:
		return storageKeyPrefixQueuedTask
	case taskStateRunning:
		return storageKeyPrefixRunningTask
	case taskStateCompleted:
		return storageKeyPrefixCompletedTask
	default:
		panic(fmt.Sprintf("unexpected task state %q", state))
	}
}

func taskStatusState(status taskStatus) taskState {
	switch {
	case status == taskStatusQueued:
		return taskStateQueued
	case status == taskStatusRunning:
		return taskStateRunning
	case isCompletedTaskStatus(status):
		return taskStateCompleted
	default:
		panic(fmt.Sprintf("unexpected task status %q", status))
	}
}

func isCompletedTaskStatus(status taskStatus) bool {
	for _, s := range taskStateStatusesCompleted {
		if s == status {
			return true
		}
	}

	return false
}

func taskLogStorageKey(uuid string) string {
	return storageKeyPrefixTaskLog + uuid
}

func storageEntryToTask(entry *logical.StorageEntry) (*Task, error) {
	var task *Task
	if err := json.Unmarshal(entry.Value, &task); err != nil {
		return nil, err
	}

	return task, nil
}
