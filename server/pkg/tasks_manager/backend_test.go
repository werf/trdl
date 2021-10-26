package tasks_manager

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
	"github.com/werf/logboek"
)

const randomUUID = "bfc441c7-a143-4ab2-9aac-4d109cef5018"

func TestManager_pathConfigureCreateOrUpdate(t *testing.T) {
	for _, op := range []logical.Operation{logical.CreateOperation, logical.UpdateOperation} {
		t.Run(string(op), func(t *testing.T) {
			t.Run("default", func(t *testing.T) {
				ctx, b, _, storage := pathTestSetup(t)

				req := &logical.Request{
					Operation: logical.CreateOperation,
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
		expectedTimeout := 50 * time.Hour
		expectedHistoryLimit := 1000
		expectedConfig := &configuration{
			TaskTimeout:      expectedTimeout,
			TaskHistoryLimit: expectedHistoryLimit,
		}
		expectedResponseData := map[string]interface{}{
			fieldNameTaskTimeout:      expectedTimeout / time.Second,
			fieldNameTaskHistoryLimit: expectedHistoryLimit,
		}

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
			assert.Equal(t, map[string]interface{}{}, resp.Data)
		}
	})

	t.Run("all", func(t *testing.T) {
		queuedTaskUUID := assertAndAddNewTaskToStorage(t, ctx, storage)
		runningTaskUUID := assertAndAddRunningTaskToStorage(t, ctx, storage)

		req := &logical.Request{
			Operation: logical.ReadOperation,
			Path:      "task",
			Data:      make(map[string]interface{}),
			Storage:   storage,
		}

		resp, err := b.HandleRequest(ctx, req)
		assert.Nil(t, err)
		if assert.NotNil(t, resp) {
			assert.Equal(t, map[string]interface{}{"keys": []string{runningTaskUUID, queuedTaskUUID}}, resp.Data)
		}
	})
}

func TestManager_pathTaskStatus(t *testing.T) {
	ctx, b, _, storage := pathTestSetup(t)

	t.Run("nonexistent", func(t *testing.T) {
		req := &logical.Request{
			Operation: logical.ReadOperation,
			Path:      "task/" + randomUUID,
			Data:      make(map[string]interface{}),
			Storage:   storage,
		}

		resp, err := b.HandleRequest(ctx, req)
		assert.Nil(t, err)
		assert.Equal(t, logical.ErrorResponse("Task %q not found", randomUUID), resp)
	})

	// fixtures
	queuedTaskUUID := assertAndAddNewTaskToStorage(t, ctx, storage)
	runningTaskUUID := assertAndAddRunningTaskToStorage(t, ctx, storage)

	for _, test := range []struct {
		taskState taskState
		taskUUID  string
	}{
		{
			taskState: taskStateQueued,
			taskUUID:  queuedTaskUUID,
		},
		{
			taskState: taskStateRunning,
			taskUUID:  runningTaskUUID,
		},
	} {
		t.Run(string(test.taskState), func(t *testing.T) {
			req := &logical.Request{
				Operation: logical.ReadOperation,
				Path:      "task/" + test.taskUUID,
				Data:      make(map[string]interface{}),
				Storage:   storage,
			}

			testTask, err := getTaskFromStorage(ctx, storage, test.taskState, test.taskUUID)
			assert.Nil(t, err)
			assert.NotNil(t, testTask)

			resp, err := b.HandleRequest(ctx, req)
			assert.Nil(t, err)
			if assert.NotNil(t, resp) {
				expectedResponseData := map[string]interface{}{
					"uuid":     testTask.UUID,
					"status":   testTask.Status,
					"reason":   testTask.Reason,
					"created":  testTask.Created,
					"modified": testTask.Modified,
				}

				assert.Equal(t, expectedResponseData, resp.Data)
			}
		})
	}
}

func TestManager_pathTaskCancel(t *testing.T) {
	ctx, b, m, storage := pathTestSetup(t)

	t.Run("nonexistent", func(t *testing.T) {
		req := &logical.Request{
			Operation: logical.CreateOperation,
			Path:      "task/" + randomUUID + "/cancel",
			Data:      make(map[string]interface{}),
			Storage:   storage,
		}

		resp, err := b.HandleRequest(ctx, req)
		assert.Nil(t, err)
		if assert.NotNil(t, resp) {
			assert.Equal(t, []string{"task \"bfc441c7-a143-4ab2-9aac-4d109cef5018\" not running"}, resp.Warnings)
		}
	})

	t.Run(string(taskStateRunning), func(t *testing.T) {
		startedCh := make(chan bool)
		taskFunc := testTaskAction(startedCh)

		uuid, err := m.AddTask(ctx, storage, taskFunc)
		assert.Nil(t, err)
		assert.NotEmpty(t, uuid)

		// wait till the task is started
		<-startedCh

		req := &logical.Request{
			Operation: logical.CreateOperation,
			Path:      "task/" + uuid + "/cancel",
			Data:      make(map[string]interface{}),
			Storage:   storage,
		}

		resp, err := b.HandleRequest(ctx, req)
		assert.Nil(t, err)
		assert.Nil(t, resp)
	})
}

func TestManager_pathTaskLog(t *testing.T) {
	ctx, b, m, storage := pathTestSetup(t)

	t.Run("nonexistent", func(t *testing.T) {
		req := &logical.Request{
			Operation: logical.ReadOperation,
			Path:      "task/" + randomUUID + "/log",
			Data:      make(map[string]interface{}),
			Storage:   storage,
		}

		resp, err := b.HandleRequest(ctx, req)
		assert.Nil(t, err)
		assert.Equal(t, logical.ErrorResponse("Task %q not found", randomUUID), resp)
	})

	t.Run(string(taskStateQueued), func(t *testing.T) {
		queuedTaskUUID := assertAndAddNewTaskToStorage(t, ctx, storage)

		req := &logical.Request{
			Operation: logical.ReadOperation,
			Path:      "task/" + queuedTaskUUID + "/log",
			Data:      make(map[string]interface{}),
			Storage:   storage,
		}

		resp, err := b.HandleRequest(ctx, req)
		assert.Nil(t, err)
		assert.Equal(t, logical.ErrorResponse("Task %q in queue", queuedTaskUUID), resp)
	})

	t.Run(string(taskStateCompleted), func(t *testing.T) {
		expectedLog := "hello world!"
		queuedTaskUUID := assertAndAddCompletedTaskToStorage(t, ctx, storage, taskStatusSucceeded, switchTaskToCompletedInStorageOptions{
			log: []byte(expectedLog),
		})

		req := &logical.Request{
			Operation: logical.ReadOperation,
			Path:      "task/" + queuedTaskUUID + "/log",
			Data:      make(map[string]interface{}),
			Storage:   storage,
		}

		resp, err := b.HandleRequest(ctx, req)
		assert.Nil(t, err)
		if assert.NotNil(t, resp) {
			assert.Equal(t, map[string]interface{}{"result": expectedLog}, resp.Data)
		}
	})

	t.Run(string(taskStateRunning), func(t *testing.T) {
		msgCh := make(chan string)
		msgSentCh := make(chan bool)
		uuid, err := m.RunTask(ctx, storage, taskActionWithLogCh(msgCh, msgSentCh))
		assert.Nil(t, err)

		var expectedLog string
		for _, msg := range []string{"", "hello ", "world!"} {
			expectedLog += msg

			// send log message
			msgCh <- msg
			<-msgSentCh

			req := &logical.Request{
				Operation: logical.ReadOperation,
				Path:      "task/" + uuid + "/log",
				Data:      make(map[string]interface{}),
				Storage:   storage,
			}

			resp, err := b.HandleRequest(ctx, req)
			assert.Nil(t, err)
			if assert.NotNil(t, resp) {
				assert.Equal(t, map[string]interface{}{"result": expectedLog}, resp.Data)
			}
		}
	})

	t.Run("offset and limit", func(t *testing.T) {
		for _, test := range []struct {
			name           string
			offset         int
			limit          int
			expectedErrMsg string
		}{
			{
				name:           "negative offset",
				offset:         -1,
				expectedErrMsg: "Field \"offset\" cannot be negative",
			},
			{
				name:           "negative limit",
				limit:          -1,
				expectedErrMsg: "Field \"limit\" cannot be negative",
			},
		} {
			t.Run(test.name, func(t *testing.T) {
				req := &logical.Request{
					Operation: logical.ReadOperation,
					Path:      "task/" + randomUUID + "/log",
					Data: map[string]interface{}{
						fieldNameOffset: test.offset,
						fieldNameLimit:  test.limit,
					},
					Storage: storage,
				}

				resp, err := b.HandleRequest(ctx, req)
				assert.Nil(t, err)
				assert.Equal(t, logical.ErrorResponse(test.expectedErrMsg), resp)
			})
		}

		expectedLog := "l" + strings.Repeat("o", 256*256) + "ng string"
		queuedTaskUUID := assertAndAddCompletedTaskToStorage(t, ctx, storage, taskStatusSucceeded, switchTaskToCompletedInStorageOptions{
			log: []byte(expectedLog),
		})

		for _, test := range []struct {
			name        string
			offset      *int
			limit       *int
			expectedLog string
		}{
			{
				name:        "default offset and limit",
				expectedLog: expectedLog[:fieldDefaultLimit],
			},
			{
				name:        "custom limit",
				limit:       getIntPointer(15),
				expectedLog: expectedLog[:15],
			},
			{
				name:        "no limit",
				limit:       getIntPointer(0),
				expectedLog: expectedLog,
			},
			{
				name:        "offset exceeding the size of the log",
				offset:      getIntPointer(len(expectedLog)),
				expectedLog: "",
			},
		} {
			t.Run(test.name, func(t *testing.T) {
				data := map[string]interface{}{}
				if test.limit != nil {
					data[fieldNameLimit] = test.limit
				}

				if test.offset != nil {
					data[fieldNameOffset] = test.offset
				}

				req := &logical.Request{
					Operation: logical.ReadOperation,
					Path:      "task/" + queuedTaskUUID + "/log",
					Data:      data,
					Storage:   storage,
				}

				resp, err := b.HandleRequest(ctx, req)
				assert.Nil(t, err)
				if assert.NotNil(t, resp) && assert.NotNil(t, resp.Data["result"]) {
					assert.Len(t, resp.Data["result"], len(test.expectedLog))
					assert.Equal(t, resp.Data["result"], test.expectedLog)
				}
			})
		}
	})
}

func getIntPointer(val int) *int {
	return &val
}

func assertAndAddNewTaskToStorage(t *testing.T, ctx context.Context, storage logical.Storage) string {
	taskUUID, err := addNewTaskToStorage(ctx, storage)
	assert.Nil(t, err)
	assert.NotEmpty(t, taskUUID)

	task, err := getTaskFromStorage(ctx, storage, taskStateQueued, taskUUID)
	assert.Nil(t, err)
	assert.NotNil(t, task)

	return taskUUID
}

func assertAndAddRunningTaskToStorage(t *testing.T, ctx context.Context, storage logical.Storage) string {
	taskUUID := assertAndAddNewTaskToStorage(t, ctx, storage)

	err := switchTaskToRunningInStorage(ctx, storage, taskUUID)
	assert.Nil(t, err)

	task, err := getTaskFromStorage(ctx, storage, taskStateQueued, taskUUID)
	assert.Nil(t, err)
	assert.Nil(t, task, "queued task must be deleted from storage")

	task, err = getTaskFromStorage(ctx, storage, taskStateRunning, taskUUID)
	assert.Nil(t, err)
	assert.NotNil(t, task)

	return taskUUID
}

func assertAndAddCompletedTaskToStorage(t *testing.T, ctx context.Context, storage logical.Storage, status taskStatus, opts switchTaskToCompletedInStorageOptions) string {
	taskUUID := assertAndAddRunningTaskToStorage(t, ctx, storage)

	err := switchTaskToCompletedInStorage(ctx, storage, status, taskUUID, opts)
	assert.Nil(t, err)

	task, err := getTaskFromStorage(ctx, storage, taskStateRunning, taskUUID)
	assert.Nil(t, err)
	assert.Nil(t, task, "running task must be deleted from storage")

	task, err = getTaskFromStorage(ctx, storage, taskStateCompleted, taskUUID)
	assert.Nil(t, err)
	if assert.NotNil(t, task) {
		assert.Equal(t, string(status), task.Status)
	}

	return taskUUID
}

func pathTestSetup(t *testing.T) (context.Context, logical.Backend, *Manager, logical.Storage) {
	ctx := context.Background()
	m := NewManager(hclog.L()) // TODO: use worker interface
	storage := &logical.InmemStorage{}

	config := logical.TestBackendConfig()
	config.StorageView = storage

	b := &framework.Backend{Paths: m.Paths()}
	err := b.Setup(ctx, config)
	assert.Nil(t, err)

	return ctx, b, m, storage
}

func testTaskAction(startedCh chan bool) func(context.Context, logical.Storage) error {
	return func(context.Context, logical.Storage) error {
		startedCh <- true
		select {}
	}
}

func taskActionWithLogCh(msgCh chan string, msgSentCh chan bool) func(context.Context, logical.Storage) error {
	return func(ctx context.Context, _ logical.Storage) error {
		for {
			logboek.Context(ctx).Log(<-msgCh)
			msgSentCh <- true
		}
	}
}
