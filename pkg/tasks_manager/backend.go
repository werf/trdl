package tasks_manager

import (
	"context"
	"fmt"
	"time"

	"github.com/fatih/structs"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/tasks_manager/worker"
)

const (
	fieldNameTaskTimeout      = "task_timeout"
	fieldNameTaskHistoryLimit = "task_history_limit"
	fieldNameUUID             = "uuid"
	fieldNameLimit            = "limit"
	fieldNameOffset           = "offset"

	fieldDefaultTaskTimeout      = "10m"
	fieldDefaultTaskHistoryLimit = 10

	defaultTaskTimeoutDuration = 10 * time.Minute
)

var (
	pathPatternConfigure  = "task/configure/?"
	pathPatternTaskList   = "task/?"
	pathPatternTaskStatus = "task/" + uuidPattern(fieldNameUUID) + "$"
	pathPatternTaskCancel = "task/" + uuidPattern(fieldNameUUID) + "/cancel$"
	pathPatternTaskLog    = "task/" + uuidPattern(fieldNameUUID) + "/log$"

	errorResponseConfigurationNotFound = logical.ErrorResponse("configuration not found")
)

func (m *Manager) Paths() []*framework.Path {
	return []*framework.Path{
		{
			Pattern: pathPatternConfigure,
			Fields: map[string]*framework.FieldSchema{
				fieldNameTaskTimeout: {
					Type:    framework.TypeDurationSecond,
					Default: fieldDefaultTaskTimeout,
				},
				fieldNameTaskHistoryLimit: {
					Type:    framework.TypeInt,
					Default: fieldDefaultTaskHistoryLimit,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: m.pathConfigureCreateOrUpdate,
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: m.pathConfigureCreateOrUpdate,
				},
				logical.ReadOperation: &framework.PathOperation{
					Callback: m.pathConfigureRead,
				},
			},
		},
		{
			Pattern: pathPatternTaskList,
			Fields:  map[string]*framework.FieldSchema{},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: m.pathTaskList,
				},
			},
		},
		{
			Pattern: pathPatternTaskStatus,
			Fields: map[string]*framework.FieldSchema{
				fieldNameUUID: {
					Type:     framework.TypeNameString,
					Required: true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: m.pathTaskStatus,
				},
			},
		},
		{
			Pattern: pathPatternTaskCancel,
			Fields: map[string]*framework.FieldSchema{
				fieldNameUUID: {
					Type:     framework.TypeNameString,
					Required: true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: m.pathTaskCancel,
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: m.pathTaskCancel,
				},
			},
		},
		{
			Pattern: pathPatternTaskLog,
			Fields: map[string]*framework.FieldSchema{
				fieldNameUUID: {
					Type:     framework.TypeNameString,
					Required: true,
				},
				fieldNameLimit: {
					Type:    framework.TypeInt,
					Default: 500,
				},
				fieldNameOffset: {
					Type:    framework.TypeInt,
					Default: 0,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Callback: m.pathTaskLogRead,
				},
				logical.UpdateOperation: &framework.PathOperation{
					Callback: m.pathTaskLogRead,
				},
			},
		},
	}
}

func (m *Manager) pathConfigureCreateOrUpdate(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	taskTimeout := time.Duration(fields.Get(fieldNameTaskTimeout).(int)) * time.Second
	taskHistoryLimit := fields.Get(fieldNameTaskHistoryLimit).(int)

	cfg := &configuration{
		TaskTimeout:      taskTimeout,
		TaskHistoryLimit: taskHistoryLimit,
	}

	if err := putConfiguration(ctx, req.Storage, cfg); err != nil {
		return logical.ErrorResponse(fmt.Sprintf("unable to save configuration: %s", err)), nil
	}

	return nil, nil
}

func (m *Manager) pathConfigureRead(ctx context.Context, req *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	c, err := getConfiguration(ctx, req.Storage)
	if err != nil {
		return logical.ErrorResponse(fmt.Sprintf("unable to get configuration: %s", err)), nil
	}

	if c == nil {
		return errorResponseConfigurationNotFound, nil
	}

	data := structs.Map(c)
	data[fieldNameTaskTimeout] = c.TaskTimeout.String()
	return &logical.Response{Data: data}, nil
}

func (m *Manager) pathTaskList(ctx context.Context, req *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	queuedTaskList, err := req.Storage.List(ctx, storageKeyPrefixQueuedTask)
	if err != nil {
		return nil, err
	}

	otherTaskList, err := req.Storage.List(ctx, storageKeyPrefixTask)
	if err != nil {
		return nil, err
	}

	return &logical.Response{
		Data: map[string]interface{}{
			"uuids": append(otherTaskList, queuedTaskList...),
		},
	}, nil
}

func (m *Manager) pathTaskStatus(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	uuid := fields.Get(fieldNameUUID).(string)

	var task *Task
	for _, getTaskFromStorageFunc := range []func(ctx context.Context, storage logical.Storage, uuid string) (*Task, error){
		getQueuedTaskFromStorage,
		getTaskFromStorage,
	} {
		t, err := getTaskFromStorageFunc(ctx, req.Storage, uuid)
		if err != nil {
			return nil, err
		}

		if t != nil {
			task = t
			break
		}
	}

	if task == nil {
		return logical.ErrorResponse(fmt.Sprintf("task %q not found", uuid)), nil
	}

	return &logical.Response{
		Data: map[string]interface{}{
			"status": structs.Map(task),
		},
	}, nil
}

func (m *Manager) pathTaskCancel(_ context.Context, _ *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	uuid := fields.Get(fieldNameUUID).(string)

	if canceled := m.Worker.CancelRunningJobByTaskUUID(uuid); !canceled {
		return &logical.Response{
			Warnings: []string{
				fmt.Sprintf("task %q not running", uuid),
			},
		}, nil
	}

	return nil, nil
}

func (m *Manager) pathTaskLogRead(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	offset := fields.Get(fieldNameOffset).(int)
	limit := fields.Get(fieldNameLimit).(int)
	uuid := fields.Get(fieldNameUUID).(string)

	if offset < 0 {
		return logical.ErrorResponse("field %q cannot be negative", fieldNameOffset), nil
	}

	if limit < 0 {
		return logical.ErrorResponse("field %q cannot be negative", fieldNameLimit), nil
	}

	data, resp, err := m.readTaskLog(ctx, req.Storage, uuid)
	if err != nil {
		return nil, err
	} else if resp != nil {
		return resp, nil
	}

	if len(data) < offset {
		data = nil
	} else if len(data[offset:]) < limit || limit == 0 {
		data = data[offset:]
	} else {
		data = data[offset : offset+limit]
	}

	return &logical.Response{
		Data: map[string]interface{}{
			"log": string(data),
		},
	}, nil
}

func (m *Manager) readTaskLog(ctx context.Context, reqStorage logical.Storage, uuid string) ([]byte, *logical.Response, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// check task existence
	{
		task, err := getTaskFromStorage(ctx, reqStorage, uuid)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to get task %q from storage: %s", uuid, err)
		}

		if task == nil {
			queuedTask, err := getQueuedTaskFromStorage(ctx, reqStorage, uuid)
			if err != nil {
				return nil, nil, fmt.Errorf("unable to get queued task %q from storage: %s", uuid, err)
			}

			if queuedTask != nil {
				return nil, logical.ErrorResponse(fmt.Sprintf("task %q in queue", uuid)), nil
			} else {
				return nil, logical.ErrorResponse(fmt.Sprintf("task %q not found", uuid)), nil
			}
		}
	}

	// try to get running task log
	var data []byte
	withHold := m.Worker.HoldRunningJobByTaskUUID(uuid, func(job *worker.Job) {
		data = job.Log()
	})

	if withHold {
		return data, nil, nil
	}

	// get task log from storage
	data, err := getTaskLogFromStorage(ctx, reqStorage, uuid)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to get task log %q from storage: %s", uuid, err)
	}

	return data, nil, nil
}

const uuidPatternRegexp = "(?i:[0-9A-F]{8}-[0-9A-F]{4}-[4][0-9A-F]{3}-[89AB][0-9A-F]{3}-[0-9A-F]{12})"

func uuidPattern(name string) string {
	return fmt.Sprintf(`(?P<%s>%s)`, name, uuidPatternRegexp)
}
