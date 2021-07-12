package tasks_manager

import (
	"context"
	"math/rand"
	"sort"
	"testing"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
)

// check that Manager.PeriodicTask deletes completed tasks from storage as configured
func TestManager_PeriodicTask(t *testing.T) {
	testFunc := func(ctx context.Context, m Interface, storage logical.Storage, testedTaskHistoryLimit, numberOfPopulatedTasks int) {
		// populate storage with tasks
		for i := 0; i < numberOfPopulatedTasks; i++ {
			_ = assertAndAddNewTaskToStorage(t, ctx, storage)
			_ = assertAndAddRunningTaskToStorage(t, ctx, storage)
			_ = assertAndAddCompletedTaskToStorage(t, ctx, storage, taskStateStatusesCompleted[rand.Intn(len(taskStateStatusesCompleted))], switchTaskToCompletedInStorageOptions{log: []byte("test")})
		}

		// prepare expected slices
		var expectedQueuedTaskUUIDs []string
		var expectedRunningTaskUUIDs []string
		var expectedCompletedTaskUUIDs []string
		var expectedTaskLogUUIDs []string
		{
			queuedTaskUUIDs, err := storage.List(ctx, taskStorageKeyPrefix(taskStateQueued))
			assert.Nil(t, err)
			assert.Len(t, queuedTaskUUIDs, numberOfPopulatedTasks)

			expectedQueuedTaskUUIDs = queuedTaskUUIDs

			runningTaskUUIDs, err := storage.List(ctx, taskStorageKeyPrefix(taskStateRunning))
			assert.Nil(t, err)
			assert.Len(t, runningTaskUUIDs, numberOfPopulatedTasks)

			expectedRunningTaskUUIDs = runningTaskUUIDs

			taskLogUUIDs, err := storage.List(ctx, storageKeyPrefixTaskLog)
			assert.Nil(t, err)
			assert.Len(t, taskLogUUIDs, numberOfPopulatedTasks)

			completedTaskUUIDs, err := storage.List(ctx, taskStorageKeyPrefix(taskStateCompleted))
			assert.Nil(t, err)
			assert.Len(t, completedTaskUUIDs, numberOfPopulatedTasks)

			var completedTasks []*Task
			for _, uuid := range completedTaskUUIDs {
				task, err := getTaskFromStorage(ctx, storage, taskStateCompleted, uuid)
				assert.Nil(t, err)
				completedTasks = append(completedTasks, task)
			}

			sort.Slice(completedTasks, func(i, j int) bool {
				return completedTasks[i].Modified.After(completedTasks[j].Modified)
			})

			if len(completedTasks) > testedTaskHistoryLimit {
				completedTasks = append([]*Task(nil), completedTasks[:testedTaskHistoryLimit]...)
			}

			for _, task := range completedTasks {
				expectedCompletedTaskUUIDs = append(expectedCompletedTaskUUIDs, task.UUID)
			}

			expectedTaskLogUUIDs = expectedCompletedTaskUUIDs
		}

		req := &logical.Request{
			Operation:  logical.ReadOperation,
			Path:       "",
			Data:       make(map[string]interface{}),
			Storage:    storage,
			Connection: &logical.Connection{},
		}

		err := m.PeriodicTask(ctx, req)
		assert.Nil(t, err)

		// check storage
		{
			queuedTaskUUIDs, err := storage.List(ctx, taskStorageKeyPrefix(taskStateQueued))
			assert.Nil(t, err)
			assert.Equal(t, expectedQueuedTaskUUIDs, queuedTaskUUIDs)

			runningTaskUUIDs, err := storage.List(ctx, taskStorageKeyPrefix(taskStateRunning))
			assert.Nil(t, err)
			assert.Equal(t, expectedRunningTaskUUIDs, runningTaskUUIDs)

			completedTaskUUIDs, err := storage.List(ctx, taskStorageKeyPrefix(taskStateCompleted))
			sort.Strings(expectedCompletedTaskUUIDs)
			sort.Strings(completedTaskUUIDs)
			assert.Nil(t, err)
			assert.Equal(t, expectedCompletedTaskUUIDs, completedTaskUUIDs)

			taskLogUUIDs, err := storage.List(ctx, storageKeyPrefixTaskLog)
			sort.Strings(taskLogUUIDs)
			sort.Strings(expectedTaskLogUUIDs)
			assert.Nil(t, err)
			assert.Equal(t, expectedTaskLogUUIDs, taskLogUUIDs)
		}
	}

	t.Run("default task_history_limit", func(t *testing.T) {
		ctx, m, storage := setupTest()
		testFunc(ctx, m, storage, fieldDefaultTaskHistoryLimit, fieldDefaultTaskHistoryLimit+5)
	})

	t.Run("custom task_history_limit", func(t *testing.T) {
		ctx, m, storage := setupTest()
		customTaskHistoryLimit := 3

		// set up task history limit
		{
			cfg := &configuration{
				TaskHistoryLimit: customTaskHistoryLimit,
			}

			err := putConfiguration(ctx, storage, cfg)
			assert.Nil(t, err)
		}

		testFunc(ctx, m, storage, customTaskHistoryLimit, customTaskHistoryLimit+10)
	})
}
