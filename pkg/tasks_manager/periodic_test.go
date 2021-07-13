package tasks_manager

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"testing"
	"time"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type PeriodicTaskSuite struct {
	suite.Suite
}

func (suite *PeriodicTaskSuite) TestCleanupTaskHistory() {
	suite.T().Run("default task_history_limit", func(t *testing.T) {
		ctx, m, storage := setupTest()
		suite.testCleanupTaskHistory(ctx, m, storage, fieldDefaultTaskHistoryLimit)
	})

	suite.T().Run("custom task_history_limit", func(t *testing.T) {
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

		suite.testCleanupTaskHistory(ctx, m, storage, customTaskHistoryLimit)
	})
}

func (suite *PeriodicTaskSuite) testCleanupTaskHistory(ctx context.Context, m Interface, storage logical.Storage, testedTaskHistoryLimit int) {
	numberOfPopulatedTasks := testedTaskHistoryLimit + 10

	// populate storage with tasks and define expectations
	var expectedQueuedTaskUUIDs []string
	var expectedRunningTaskUUIDs []string
	var expectedCompletedTaskUUIDs []string
	var expectedTaskLogUUIDs []string
	{
		for i := 0; i < numberOfPopulatedTasks; i++ {
			_ = assertAndAddNewTaskToStorage(suite.T(), ctx, storage)
			_ = assertAndAddRunningTaskToStorage(suite.T(), ctx, storage)
			_ = assertAndAddCompletedTaskToStorage(suite.T(), ctx, storage, taskStateStatusesCompleted[rand.Intn(len(taskStateStatusesCompleted))], switchTaskToCompletedInStorageOptions{log: []byte("test")})
		}

		{
			queuedTaskUUIDs, err := storage.List(ctx, taskStorageKeyPrefix(taskStateQueued))
			assert.Nil(suite.T(), err)
			assert.Len(suite.T(), queuedTaskUUIDs, numberOfPopulatedTasks)

			expectedQueuedTaskUUIDs = queuedTaskUUIDs

			runningTaskUUIDs, err := storage.List(ctx, taskStorageKeyPrefix(taskStateRunning))
			assert.Nil(suite.T(), err)
			assert.Len(suite.T(), runningTaskUUIDs, numberOfPopulatedTasks)

			expectedRunningTaskUUIDs = runningTaskUUIDs

			taskLogUUIDs, err := storage.List(ctx, storageKeyPrefixTaskLog)
			assert.Nil(suite.T(), err)
			assert.Len(suite.T(), taskLogUUIDs, numberOfPopulatedTasks)

			completedTaskUUIDs, err := storage.List(ctx, taskStorageKeyPrefix(taskStateCompleted))
			assert.Nil(suite.T(), err)
			assert.Len(suite.T(), completedTaskUUIDs, numberOfPopulatedTasks)

			var completedTasks []*Task
			for _, uuid := range completedTaskUUIDs {
				task, err := getTaskFromStorage(ctx, storage, taskStateCompleted, uuid)
				assert.Nil(suite.T(), err)
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
	}

	req := &logical.Request{Storage: storage}
	err := m.PeriodicTask(ctx, req)
	assert.Nil(suite.T(), err)

	// check storage
	{
		queuedTaskUUIDs, err := storage.List(ctx, taskStorageKeyPrefix(taskStateQueued))
		assert.Nil(suite.T(), err)
		assert.Equal(suite.T(), expectedQueuedTaskUUIDs, queuedTaskUUIDs)

		runningTaskUUIDs, err := storage.List(ctx, taskStorageKeyPrefix(taskStateRunning))
		assert.Nil(suite.T(), err)
		assert.Equal(suite.T(), expectedRunningTaskUUIDs, runningTaskUUIDs)

		completedTaskUUIDs, err := storage.List(ctx, taskStorageKeyPrefix(taskStateCompleted))
		sort.Strings(expectedCompletedTaskUUIDs)
		sort.Strings(completedTaskUUIDs)
		assert.Nil(suite.T(), err)
		assert.Equal(suite.T(), expectedCompletedTaskUUIDs, completedTaskUUIDs)

		taskLogUUIDs, err := storage.List(ctx, storageKeyPrefixTaskLog)
		sort.Strings(taskLogUUIDs)
		sort.Strings(expectedTaskLogUUIDs)
		assert.Nil(suite.T(), err)
		assert.Equal(suite.T(), expectedTaskLogUUIDs, taskLogUUIDs)
	}
}

func (suite *PeriodicTaskSuite) TestLastPeriodicRunTimestamp() {
	for _, test := range []struct {
		name                            string
		initialLastPeriodicRunTimestamp interface{}
		expectedChanged                 bool
	}{
		{
			name:                            "none",
			initialLastPeriodicRunTimestamp: nil,
			expectedChanged:                 true,
		},
		{
			name:                            "less than periodicTaskPeriod",
			initialLastPeriodicRunTimestamp: (time.Now().Add(-periodicTaskPeriod + time.Minute)).Unix(),
			expectedChanged:                 false,
		},
		{
			name:                            "more than periodicTaskPeriod",
			initialLastPeriodicRunTimestamp: (time.Now().Add(-periodicTaskPeriod - time.Minute)).Unix(),
			expectedChanged:                 true,
		},
	} {
		suite.T().Run(test.name, func(t *testing.T) {
			ctx, m, s := setupTest()

			var initialValue []byte
			if test.initialLastPeriodicRunTimestamp != nil {
				initialValue = []byte(fmt.Sprintf("%d", test.initialLastPeriodicRunTimestamp))
				entry := &logical.StorageEntry{Key: storageKeyLastPeriodicRunTimestamp, Value: initialValue}
				err := s.Put(ctx, entry)
				assert.Nil(t, err)
			}

			req := &logical.Request{Storage: s}
			err := m.PeriodicTask(ctx, req)
			assert.Nil(t, err)

			entry, err := s.Get(ctx, storageKeyLastPeriodicRunTimestamp)
			assert.Nil(t, err)
			assert.NotNil(t, entry)

			if test.expectedChanged {
				assert.NotEqual(t, string(initialValue), string(entry.Value))
			} else {
				assert.Equal(t, string(initialValue), string(entry.Value))
			}
		})
	}
}

func TestManager_PeriodicTask(t *testing.T) {
	suite.Run(t, new(PeriodicTaskSuite))
}
