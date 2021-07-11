package tasks_manager

import (
	"encoding/json"
	"time"

	"github.com/hashicorp/vault/sdk/logical"
	uuid "github.com/satori/go.uuid"
)

type Task struct {
	UUID     string
	Status   string
	Reason   string
	Created  time.Time
	Modified time.Time
}

func newTask() *Task {
	task := &Task{}
	task.UUID = uuid.NewV4().String()
	task.Status = taskStatusQueued

	tNow := time.Now()
	task.Created = tNow
	task.Modified = tNow

	return task
}

func queuedTaskToStorageEntry(task *Task) (*logical.StorageEntry, error) {
	return logical.StorageEntryJSON(queuedTaskStorageKey(task.UUID), task)
}

func taskToStorageEntry(task *Task) (*logical.StorageEntry, error) {
	return logical.StorageEntryJSON(taskStorageKey(task.UUID), task)
}

func storageEntryToTask(entry *logical.StorageEntry) (*Task, error) {
	var task *Task
	if err := json.Unmarshal(entry.Value, &task); err != nil {
		return nil, err
	}

	return task, nil
}

func queuedTaskStorageKey(uuid string) string {
	return storageKeyPrefixQueuedTask + uuid
}

func taskStorageKey(uuid string) string {
	return storageKeyPrefixTask + uuid
}

func taskLogStorageKey(uuid string) string {
	return storageKeyPrefixTaskLog + uuid
}
