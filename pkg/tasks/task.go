package tasks

import (
	"encoding/json"
	"time"

	"github.com/hashicorp/vault/sdk/logical"
)

type Task struct {
	UUID     string
	Status   string
	Reason   string
	Created  time.Time
	Modified time.Time
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

func taskStorageKey(uuid string) string {
	return storageKeyPrefixTask + uuid
}

func taskLogStorageKey(uuid string) string {
	return storageKeyPrefixTaskLog + uuid
}
