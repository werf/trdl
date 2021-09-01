package tasks_manager

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/hashicorp/vault/sdk/logical"
)

const storageKeyLastPeriodicRunTimestamp = "tasks_manager_last_periodic_run_timestamp"

var periodicTaskPeriod = time.Hour

func (m *Manager) PeriodicFunc(ctx context.Context, req *logical.Request) error {
	// lock manager
	m.mu.Lock()
	defer m.mu.Unlock()

	// skip if the time since the last successfully passed periodic task less than period of the periodic task (1 hour)
	{
		entry, err := req.Storage.Get(ctx, storageKeyLastPeriodicRunTimestamp)
		if err != nil {
			return fmt.Errorf("unable to get %q from storage: %s", storageKeyLastPeriodicRunTimestamp, err)
		}

		if entry != nil {
			lastRunTimestamp, err := strconv.ParseInt(string(entry.Value), 10, 64)
			if err == nil && time.Since(time.Unix(lastRunTimestamp, 0)) <= periodicTaskPeriod {
				return nil
			}
		}
	}

	startTime := time.Now()

	if err := m.cleanupTaskHistory(ctx, req); err != nil {
		return err
	}

	if err := req.Storage.Put(ctx, &logical.StorageEntry{Key: storageKeyLastPeriodicRunTimestamp, Value: []byte(fmt.Sprintf("%d", startTime.Unix()))}); err != nil {
		return fmt.Errorf("unable to put %q into storage: %s", storageKeyLastPeriodicRunTimestamp, err)
	}

	return nil
}

func (m *Manager) cleanupTaskHistory(ctx context.Context, req *logical.Request) error {
	// define taskHistoryLimit
	taskHistoryLimit := fieldDefaultTaskHistoryLimit
	{
		config, err := getConfiguration(ctx, req.Storage)
		if err != nil {
			return fmt.Errorf("unable to get tasks manager configuration: %s", err)
		}

		if config != nil {
			taskHistoryLimit = config.TaskHistoryLimit
		}
	}

	list, err := req.Storage.List(ctx, taskStorageKeyPrefix(taskStateCompleted))
	if err != nil {
		return err
	}

	var completedTasks []*Task
	for _, taskUUID := range list {
		task, err := getTaskFromStorage(ctx, req.Storage, taskStateCompleted, taskUUID)
		if err != nil {
			return err
		}

		completedTasks = append(completedTasks, task)
	}

	sort.Slice(completedTasks, func(i, j int) bool {
		return completedTasks[i].Modified.After(completedTasks[j].Modified)
	})

	if len(completedTasks) > taskHistoryLimit {
		completedTasks = append([]*Task(nil), completedTasks[taskHistoryLimit:]...)
	}

	for _, task := range completedTasks {
		if err := req.Storage.Delete(ctx, taskStorageKey(taskStateCompleted, task.UUID)); err != nil {
			return err
		}

		if err := req.Storage.Delete(ctx, taskLogStorageKey(task.UUID)); err != nil {
			return err
		}
	}

	return nil
}
