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
	ctx     context.Context
	manager Interface
	storage logical.Storage
}

func (suite *PeriodicTaskSuite) SetupTest() {
	suite.ctx = context.Background()
	suite.manager = initManagerWithoutWorker()
	suite.storage = &logical.InmemStorage{}
}

func (suite *PeriodicTaskSuite) TestCleanupTaskHistoryDefaultTaskHistoryLimit() {
	suite.testCleanupTaskHistory(fieldDefaultTaskHistoryLimit)
}

func (suite *PeriodicTaskSuite) TestCleanupTaskHistoryCustomTaskHistoryLimit() {
	customTaskHistoryLimit := 3

	// set up task history limit
	{
		cfg := &configuration{
			TaskHistoryLimit: customTaskHistoryLimit,
		}

		err := putConfiguration(suite.ctx, suite.storage, cfg)
		assert.Nil(suite.T(), err)
	}

	suite.testCleanupTaskHistory(customTaskHistoryLimit)
}

func (suite *PeriodicTaskSuite) testCleanupTaskHistory(testedTaskHistoryLimit int) {
	numberOfPopulatedTasks := testedTaskHistoryLimit + 10

	// populate storage with tasks and define expectations
	var expectedQueuedTaskUUIDs []string
	var expectedRunningTaskUUIDs []string
	var expectedCompletedTaskUUIDs []string
	var expectedTaskLogUUIDs []string
	{
		for i := 0; i < numberOfPopulatedTasks; i++ {
			_ = assertAndAddNewTaskToStorage(suite.T(), suite.ctx, suite.storage)
			_ = assertAndAddRunningTaskToStorage(suite.T(), suite.ctx, suite.storage)
			_ = assertAndAddCompletedTaskToStorage(suite.T(), suite.ctx, suite.storage, taskStateStatusesCompleted[rand.Intn(len(taskStateStatusesCompleted))], switchTaskToCompletedInStorageOptions{log: []byte("test")})
		}

		queuedTaskUUIDs, err := suite.storage.List(suite.ctx, taskStorageKeyPrefix(taskStateQueued))
		assert.Nil(suite.T(), err)
		assert.Len(suite.T(), queuedTaskUUIDs, numberOfPopulatedTasks)

		expectedQueuedTaskUUIDs = queuedTaskUUIDs

		runningTaskUUIDs, err := suite.storage.List(suite.ctx, taskStorageKeyPrefix(taskStateRunning))
		assert.Nil(suite.T(), err)
		assert.Len(suite.T(), runningTaskUUIDs, numberOfPopulatedTasks)

		expectedRunningTaskUUIDs = runningTaskUUIDs

		taskLogUUIDs, err := suite.storage.List(suite.ctx, storageKeyPrefixTaskLog)
		assert.Nil(suite.T(), err)
		assert.Len(suite.T(), taskLogUUIDs, numberOfPopulatedTasks)

		completedTaskUUIDs, err := suite.storage.List(suite.ctx, taskStorageKeyPrefix(taskStateCompleted))
		assert.Nil(suite.T(), err)
		assert.Len(suite.T(), completedTaskUUIDs, numberOfPopulatedTasks)

		var completedTasks []*Task
		for _, uuid := range completedTaskUUIDs {
			task, err := getTaskFromStorage(suite.ctx, suite.storage, taskStateCompleted, uuid)
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

	req := &logical.Request{Storage: suite.storage}
	err := suite.manager.PeriodicTask(suite.ctx, req)
	assert.Nil(suite.T(), err)

	// check storage
	{
		queuedTaskUUIDs, err := suite.storage.List(suite.ctx, taskStorageKeyPrefix(taskStateQueued))
		assert.Nil(suite.T(), err)
		assert.Equal(suite.T(), expectedQueuedTaskUUIDs, queuedTaskUUIDs)

		runningTaskUUIDs, err := suite.storage.List(suite.ctx, taskStorageKeyPrefix(taskStateRunning))
		assert.Nil(suite.T(), err)
		assert.Equal(suite.T(), expectedRunningTaskUUIDs, runningTaskUUIDs)

		completedTaskUUIDs, err := suite.storage.List(suite.ctx, taskStorageKeyPrefix(taskStateCompleted))
		sort.Strings(expectedCompletedTaskUUIDs)
		sort.Strings(completedTaskUUIDs)
		assert.Nil(suite.T(), err)
		assert.Equal(suite.T(), expectedCompletedTaskUUIDs, completedTaskUUIDs)

		taskLogUUIDs, err := suite.storage.List(suite.ctx, storageKeyPrefixTaskLog)
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
		suite.Run(test.name, func() {
			var initialValue []byte
			if test.initialLastPeriodicRunTimestamp != nil {
				initialValue = []byte(fmt.Sprintf("%d", test.initialLastPeriodicRunTimestamp))
				entry := &logical.StorageEntry{Key: storageKeyLastPeriodicRunTimestamp, Value: initialValue}
				err := suite.storage.Put(suite.ctx, entry)
				assert.Nil(suite.T(), err)
			}

			req := &logical.Request{Storage: suite.storage}
			err := suite.manager.PeriodicTask(suite.ctx, req)
			assert.Nil(suite.T(), err)

			entry, err := suite.storage.Get(suite.ctx, storageKeyLastPeriodicRunTimestamp)
			assert.Nil(suite.T(), err)
			assert.NotNil(suite.T(), entry)

			if test.expectedChanged {
				assert.NotEqual(suite.T(), string(initialValue), string(entry.Value))
			} else {
				assert.Equal(suite.T(), string(initialValue), string(entry.Value))
			}
		})
	}
}

func TestManager_PeriodicTask(t *testing.T) {
	suite.Run(t, new(PeriodicTaskSuite))
}
