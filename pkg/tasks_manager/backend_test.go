package tasks_manager

import (
	"context"
	"testing"
	"time"

	"github.com/fatih/structs"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
)

func TestManager_pathConfigureCreateOrUpdate(t *testing.T) {
	for _, op := range []logical.Operation{logical.CreateOperation, logical.UpdateOperation} {
		t.Run(string(op), func(t *testing.T) {
			t.Run("default", func(t *testing.T) {
				ctx, b, _, storage := pathTestSetup(t)

				req := &logical.Request{
					Operation: op,
					Path:      "task/configure",
					Data:      make(map[string]interface{}),
					Storage:   storage,
				}

				resp, err := b.HandleRequest(ctx, req)
				assert.Nil(t, err)
				assert.Nil(t, resp)

				c, err := getConfiguration(ctx, storage)
				assert.Nil(t, err)
				assert.Equal(t, &configuration{
					TaskTimeout:      defaultTaskTimeoutDuration,
					TaskHistoryLimit: fieldDefaultTaskHistoryLimit,
				}, c)
			})

			t.Run("custom", func(t *testing.T) {
				ctx, b, _, storage := pathTestSetup(t)

				expectedTaskTimeout := 5 * time.Minute
				expectedTaskHistoryLimit := 25
				fieldValueTaskTimeout := expectedTaskTimeout.String()
				fieldValueTaskHistoryLimit := expectedTaskHistoryLimit

				req := &logical.Request{
					Operation: op,
					Path:      "task/configure",
					Data: map[string]interface{}{
						fieldNameTaskTimeout:      fieldValueTaskTimeout,
						fieldNameTaskHistoryLimit: fieldValueTaskHistoryLimit,
					},
					Storage: storage,
				}

				resp, err := b.HandleRequest(ctx, req)
				assert.Nil(t, err)
				assert.Nil(t, resp)

				c, err := getConfiguration(ctx, storage)
				assert.Nil(t, err)
				assert.Equal(t, &configuration{
					TaskTimeout:      expectedTaskTimeout,
					TaskHistoryLimit: expectedTaskHistoryLimit,
				}, c)
			})
		})
	}
}

func TestManager_pathConfigureRead(t *testing.T) {
	ctx, b, _, storage := pathTestSetup(t)

	t.Run("nonexistent", func(t *testing.T) {
		req := &logical.Request{
			Operation: logical.ReadOperation,
			Path:      "task/configure",
			Data:      make(map[string]interface{}),
			Storage:   storage,
		}

		resp, err := b.HandleRequest(ctx, req)
		assert.Nil(t, err)
		assert.Equal(t, errorResponseConfigurationNotFound, resp)
	})

	t.Run("normal", func(t *testing.T) {
		expectedConfig := &configuration{
			TaskTimeout:      50 * time.Hour,
			TaskHistoryLimit: 1000,
		}
		expectedResponseData := structs.Map(expectedConfig)
		expectedResponseData[fieldNameTaskTimeout] = expectedConfig.TaskTimeout.String()

		err := putConfiguration(ctx, storage, expectedConfig)
		assert.Nil(t, err)

		req := &logical.Request{
			Operation: logical.ReadOperation,
			Path:      "task/configure",
			Data:      make(map[string]interface{}),
			Storage:   storage,
		}

		resp, err := b.HandleRequest(ctx, req)
		assert.Nil(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, expectedResponseData, resp.Data)
	})
}

func TestManager_pathTaskList(t *testing.T) {
	ctx, b, _, storage := pathTestSetup(t)

	t.Run("empty", func(t *testing.T) {
		req := &logical.Request{
			Operation: logical.ReadOperation,
			Path:      "task",
			Data:      make(map[string]interface{}),
			Storage:   storage,
		}

		resp, err := b.HandleRequest(ctx, req)
		assert.Nil(t, err)
		if assert.NotNil(t, resp) {
			assert.Equal(t, map[string]interface{}{"uuids": []string(nil)}, resp.Data)
		}
	})

	t.Run("all", func(t *testing.T) {
		// fixtures
		var runningTaskUUID string
		var queuedTaskUUID string
		{
			runningTaskUUID, queuedTaskUUID = pathTestFixtures(t, ctx, storage)
		}

		req := &logical.Request{
			Operation: logical.ReadOperation,
			Path:      "task",
			Data:      make(map[string]interface{}),
			Storage:   storage,
		}

		resp, err := b.HandleRequest(ctx, req)
		assert.Nil(t, err)
		if assert.NotNil(t, resp) {
			assert.Equal(t, map[string]interface{}{"uuids": []string{runningTaskUUID, queuedTaskUUID}}, resp.Data)
		}
	})
}

func TestManager_pathTaskStatus(t *testing.T) {
	ctx, b, _, storage := pathTestSetup(t)

	t.Run("nonexistent", func(t *testing.T) {
		randomUUID := "bfc441c7-a143-4ab2-9aac-4d109cef5018"

		req := &logical.Request{
			Operation: logical.ReadOperation,
			Path:      "task/" + randomUUID,
			Data:      make(map[string]interface{}),
			Storage:   storage,
		}

		resp, err := b.HandleRequest(ctx, req)
		assert.Nil(t, err)
		if assert.NotNil(t, resp) {
			assert.Equal(t, map[string]interface{}{"error": "task \"bfc441c7-a143-4ab2-9aac-4d109cef5018\" not found"}, resp.Data)
		}
	})

	// fixtures
	var runningTaskUUID string
	var queuedTaskUUID string
	{
		runningTaskUUID, queuedTaskUUID = pathTestFixtures(t, ctx, storage)
	}

	for _, test := range []struct {
		name        string
		taskUUID    string
		getTaskFunc func(ctx context.Context, storage logical.Storage, uuid string) (*Task, error)
	}{
		{
			name:        taskStatusQueued,
			taskUUID:    queuedTaskUUID,
			getTaskFunc: getQueuedTaskFromStorage,
		},
		{
			name:        taskStatusRunning,
			taskUUID:    runningTaskUUID,
			getTaskFunc: getTaskFromStorage,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := &logical.Request{
				Operation: logical.ReadOperation,
				Path:      "task/" + test.taskUUID,
				Data:      make(map[string]interface{}),
				Storage:   storage,
			}

			testTask, err := test.getTaskFunc(ctx, storage, test.taskUUID)
			assert.Nil(t, err)
			assert.NotNil(t, testTask)

			resp, err := b.HandleRequest(ctx, req)
			assert.Nil(t, err)
			if assert.NotNil(t, resp) {
				assert.Equal(t, map[string]interface{}{"status": structs.Map(testTask)}, resp.Data)
			}
		})
	}
}

func pathTestFixtures(t *testing.T, ctx context.Context, storage logical.Storage) (string, string) {
	runningTask := newTask()
	runningTask.Status = taskStatusRunning
	err := putTaskIntoStorage(ctx, storage, runningTask)
	assert.Nil(t, err)

	queuedTask := newTask()
	queuedTask.Status = taskStatusQueued
	err = putQueuedTaskIntoStorage(ctx, storage, queuedTask)
	assert.Nil(t, err)

	return runningTask.UUID, queuedTask.UUID
}

func pathTestSetup(t *testing.T) (context.Context, logical.Backend, Interface, logical.Storage) {
	ctx := context.Background()
	m := NewManager()
	storage := &logical.InmemStorage{}

	config := logical.TestBackendConfig()
	config.StorageView = storage

	b := &framework.Backend{Paths: m.Paths()}
	err := b.Setup(ctx, config)
	assert.Nil(t, err)

	return ctx, b, m, storage
}
