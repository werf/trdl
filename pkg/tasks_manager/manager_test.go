package tasks_manager

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
)

func TestManager_taskStartedCallback(t *testing.T) {
	ctx, m, storage := setupTest()

	assert.Nil(t, m.Storage, "should be initialized on the first action call")
	m.Storage = storage

	t.Run("nonexistent", func(t *testing.T) {
		assertPanic(
			t,
			func() { m.taskStartedCallback(ctx, "1") },
			"runtime error: the task \"1\" not found in storage",
		)
	})

	t.Run("queued", func(t *testing.T) {
		queuedTask := newTask()
		err := putQueuedTaskIntoStorage(ctx, storage, queuedTask)
		assert.Nil(t, err)

		m.taskStartedCallback(ctx, queuedTask.UUID)
		assertTaskStartedCallbackQueuedTask(t, ctx, m.Storage, queuedTask)
	})
}

func TestManager_taskCompletedCallback(t *testing.T) {
	ctx, m, storage := setupTest()

	assert.Nil(t, m.Storage, "should be initialized on the first action call")
	m.Storage = storage

	t.Run("nonexistent", func(t *testing.T) {
		assertPanic(
			t,
			func() { m.taskCompletedCallback(ctx, "1", nil) },
			"runtime error: the task \"1\" not found in storage",
		)
	})

	t.Run("running", func(t *testing.T) {
		runningTask := newTask()
		runningTask.Status = taskStatusRunning
		err := putTaskIntoStorage(ctx, storage, runningTask)
		assert.Nil(t, err)

		taskActionLog := []byte("Hello!")
		m.taskCompletedCallback(ctx, runningTask.UUID, taskActionLog)

		completed, err := getTaskFromStorage(ctx, storage, runningTask.UUID)
		assert.Nil(t, err)
		assert.Equal(t, taskStatusCompleted, completed.Status)
		assert.Empty(t, completed.Reason)

		log, err := getTaskLogFromStorage(ctx, storage, runningTask.UUID)
		assert.Nil(t, err)
		assert.Equal(t, taskActionLog, log)

		currentTaskUUID, err := getCurrentTaskUUIDFromStorage(ctx, storage)
		assert.Nil(t, err)
		assert.Empty(t, currentTaskUUID)
	})
}

func TestManager_taskFailedCallback(t *testing.T) {
	ctx, m, storage := setupTest()

	assert.Nil(t, m.Storage, "should be initialized on the first action call")
	m.Storage = storage

	t.Run("nonexistent", func(t *testing.T) {
		assertPanic(
			t,
			func() { m.taskFailedCallback(ctx, "1", nil, nil) },
			"runtime error: the task \"1\" not found in storage",
		)
	})

	t.Run("running", func(t *testing.T) {
		runningTask := newTask()
		runningTask.Status = taskStatusRunning
		err := putTaskIntoStorage(ctx, storage, runningTask)
		assert.Nil(t, err)

		taskActionErr := fmt.Errorf("error")
		taskActionLog := []byte("Hello!")
		m.taskFailedCallback(ctx, runningTask.UUID, taskActionLog, taskActionErr)

		failedTask, err := getTaskFromStorage(ctx, storage, runningTask.UUID)
		assert.Nil(t, err)
		assert.Equal(t, taskStatusFailed, failedTask.Status)
		assert.Equal(t, taskActionErr.Error(), failedTask.Reason)

		log, err := getTaskLogFromStorage(ctx, storage, runningTask.UUID)
		assert.Nil(t, err)
		assert.Equal(t, taskActionLog, log)

		currentTaskUUID, err := getCurrentTaskUUIDFromStorage(ctx, storage)
		assert.Nil(t, err)
		assert.Empty(t, currentTaskUUID)
	})
}

func assertTaskStartedCallbackQueuedTask(t *testing.T, ctx context.Context, storage logical.Storage, startedTask *Task) {
	queuedTask, err := getQueuedTaskFromStorage(ctx, storage, startedTask.UUID)
	assert.Nil(t, err)
	assert.Nil(t, queuedTask)

	runningTask, err := getTaskFromStorage(ctx, storage, startedTask.UUID)
	assert.Nil(t, err)
	assert.Equal(t, taskStatusRunning, runningTask.Status)

	currentTaskUUID, err := getCurrentTaskUUIDFromStorage(ctx, storage)
	assert.Nil(t, err)
	assert.Equal(t, startedTask.UUID, currentTaskUUID)
}

func assertPanic(t *testing.T, f func(), expectedMsg string) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		} else {
			assert.Equal(t, expectedMsg, r)
		}
	}()
	f()
}
