package queue_manager

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/hashicorp/vault/sdk/logical"
)

const storageKeyLastPeriodicRunTimestamp = "queue_manager_last_periodic_run_timestamp"

var (
	periodicTaskPeriod        = time.Hour
	periodicTaskQueuedTaskTTL = 12 * time.Hour
)

func (m *Manager) PeriodicTask(ctx context.Context, req *logical.Request) error {
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
	taskHistoryLimit := defaultTaskHistoryLimit
	{
		config, err := getConfiguration(ctx, req.Storage)
		if err != nil {
			return fmt.Errorf("unable to get queue manager configuration: %s", err)
		}

		if config != nil && config.TaskHistoryLimit != "" {
			taskHistoryLimit, err = strconv.ParseInt(config.TaskHistoryLimit, 10, 64)
			if err != nil {
				return fmt.Errorf("unexpected %q value %q: %q", fieldNameTaskHistoryLimit, config.TaskHistoryLimit, err)
			}
		}
	}

	list, err := req.Storage.List(ctx, storageKeyPrefixTask)
	if err != nil {
		return err
	}

	var queuedTasks []*Task
	var otherTasks []*Task
	for _, taskUUID := range list {
		task, err := getTaskFromStorage(ctx, req.Storage, taskUUID)
		if err != nil {
			return fmt.Errorf("unable to get task %q from storage: %s", taskUUID, err)
		}

		if task.Status == taskStatusQueued {
			queuedTasks = append(queuedTasks, task)
		} else {
			otherTasks = append(otherTasks, task)
		}
	}

	var tasksToDelete []*Task
	for _, task := range queuedTasks {
		if time.Since(task.Created) > periodicTaskQueuedTaskTTL {
			tasksToDelete = append(tasksToDelete, task)
		}
	}

	sort.Slice(otherTasks, func(i, j int) bool {
		return otherTasks[i].Created.After(otherTasks[j].Created)
	})

	if int64(len(otherTasks)) > taskHistoryLimit {
		tasksToDelete = append(tasksToDelete, otherTasks[taskHistoryLimit:]...)
	}

	for _, task := range tasksToDelete {
		if err := req.Storage.Delete(ctx, taskStorageKey(task.UUID)); err != nil {
			return err
		}

		if err := req.Storage.Delete(ctx, taskLogStorageKey(task.UUID)); err != nil {
			return err
		}
	}

	return nil
}
