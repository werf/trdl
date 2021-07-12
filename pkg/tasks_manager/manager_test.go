package tasks_manager

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestManager_taskStartedCallback(t *testing.T) {
	ctx, m, storage := setupTest()

	assert.Nil(t, m.Storage, "must be initialized on the first action call")
	m.Storage = storage

	t.Run("nonexistent", func(t *testing.T) {
		assertPanic(
			t,
			func() { m.taskStartedCallback(ctx, "1") },
			"runtime error: queued task \"1\" must be in storage",
		)
	})

	t.Run("queued", func(t *testing.T) {
		queuedTaskUUID := assertAndAddNewTaskToStorage(t, ctx, storage)

		m.taskStartedCallback(ctx, queuedTaskUUID)

		queuedTask, err := getTaskFromStorage(ctx, storage, taskStateQueued, queuedTaskUUID)
		assert.Nil(t, err)
		assert.Nil(t, queuedTask)

		runningTask, err := getTaskFromStorage(ctx, storage, taskStateRunning, queuedTaskUUID)
		assert.Nil(t, err)
		if assert.NotNil(t, runningTask) {
			assert.Equal(t, string(taskStatusRunning), runningTask.Status)
			assert.Equal(t, queuedTaskUUID, runningTask.UUID)
		}
	})
}

func TestManager_taskSucceededCallback(t *testing.T) {
	ctx, m, storage := setupTest()

	assert.Nil(t, m.Storage, "must be initialized on the first action call")
	m.Storage = storage

	t.Run("nonexistent", func(t *testing.T) {
		assertPanic(
			t,
			func() { m.taskSucceededCallback(ctx, "1", nil) },
			"runtime error: queued or running task \"1\" not found in storage",
		)
	})

	t.Run("running", func(t *testing.T) {
		runningTaskUUID := assertAndAddRunningTaskToStorage(t, ctx, storage)

		taskActionLog := []byte("Hello!")
		m.taskSucceededCallback(ctx, runningTaskUUID, taskActionLog)

		runningTask, err := getTaskFromStorage(ctx, storage, taskStateRunning, runningTaskUUID)
		assert.Nil(t, err)
		assert.Nil(t, runningTask)

		completedTask, err := getTaskFromStorage(ctx, storage, taskStateCompleted, runningTaskUUID)
		assert.Nil(t, err)
		if assert.NotNil(t, completedTask) {
			assert.Equal(t, string(taskStatusSucceeded), completedTask.Status)
			assert.Equal(t, runningTaskUUID, completedTask.UUID)
			assert.Empty(t, completedTask.Reason)
		}

		log, err := getTaskLogFromStorage(ctx, storage, runningTaskUUID)
		assert.Nil(t, err)
		assert.Equal(t, taskActionLog, log)
	})
}

func TestManager_taskFailedCallback(t *testing.T) {
	ctx, m, storage := setupTest()

	assert.Nil(t, m.Storage, "must be initialized on the first action call")
	m.Storage = storage

	taskActionErr := fmt.Errorf("error")

	t.Run("nonexistent", func(t *testing.T) {
		assertPanic(
			t,
			func() { m.taskFailedCallback(ctx, "1", nil, taskActionErr) },
			"runtime error: queued or running task \"1\" not found in storage",
		)
	})

	t.Run("running", func(t *testing.T) {
		runningTaskUUID := assertAndAddRunningTaskToStorage(t, ctx, storage)

		taskActionLog := []byte("Hello!")
		m.taskFailedCallback(ctx, runningTaskUUID, taskActionLog, taskActionErr)

		runningTask, err := getTaskFromStorage(ctx, storage, taskStateRunning, runningTaskUUID)
		assert.Nil(t, err)
		assert.Nil(t, runningTask)

		completedTask, err := getTaskFromStorage(ctx, storage, taskStateCompleted, runningTaskUUID)
		assert.Nil(t, err)
		if assert.NotNil(t, completedTask) {
			assert.Equal(t, string(taskStatusFailed), completedTask.Status)
			assert.Equal(t, runningTaskUUID, completedTask.UUID)
			assert.Equal(t, taskActionErr.Error(), completedTask.Reason)
		}

		log, err := getTaskLogFromStorage(ctx, storage, runningTaskUUID)
		assert.Nil(t, err)
		assert.Equal(t, taskActionLog, log)
	})
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
