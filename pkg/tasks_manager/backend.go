package tasks_manager

import (
	"context"
	"fmt"
	"time"

	"github.com/fatih/structs"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/tasks_manager/worker"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/util"
)

const (
	fieldNameTaskTimeout      = "task_timeout"
	fieldNameTaskHistoryLimit = "task_history_limit"
	fieldNameUUID             = "uuid"
	fieldNameLimit            = "limit"
	fieldNameOffset           = "offset"

	defaultTaskTimeoutValue    = "10m"
	defaultTaskTimeoutDuration = 10 * time.Minute
	defaultTaskHistoryLimit    = 10
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
					Default: defaultTaskTimeoutValue,
				},
				fieldNameTaskHistoryLimit: {
					Type:    framework.TypeInt,
					Default: defaultTaskHistoryLimit,
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
	resp, err := util.ValidateRequestFields(req, fields)
	if resp != nil || err != nil {
		return resp, err
	}

	var taskTimeout string
	{
		paramTaskTimeout := req.Get(fieldNameTaskTimeout)
		if paramTaskTimeout == nil {
			taskTimeout = fields.Schema[fieldNameTaskTimeout].Default.(string)
		} else {
			taskTimeout = paramTaskTimeout.(string)

			if _, err := time.ParseDuration(taskTimeout); err != nil {
				return logical.ErrorResponse(fmt.Sprintf("invalid field %q given, expected golang time duration: %s", fieldNameTaskTimeout, err)), nil
			}
		}
	}

	var taskHistoryLimit int
	{
		paramTaskHistoryLimit := req.Get(fieldNameTaskHistoryLimit)
		if paramTaskHistoryLimit == nil {
			taskHistoryLimit = fields.Schema[fieldNameTaskHistoryLimit].Default.(int)
		} else {
			taskHistoryLimit = paramTaskHistoryLimit.(int)
		}
	}

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

	return &logical.Response{Data: structs.Map(c)}, nil
}

func (m *Manager) pathTaskList(ctx context.Context, req *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	list, err := req.Storage.List(ctx, storageKeyPrefixTask)
	if err != nil {
		return nil, err
	}

	return &logical.Response{
		Data: map[string]interface{}{
			"uuids": list,
		},
	}, nil
}

func (m *Manager) pathTaskStatus(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	uuid := fields.Get(fieldNameUUID).(string)

	task, err := getTaskFromStorage(ctx, req.Storage, uuid)
	if err != nil {
		return nil, err
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

func (m *Manager) pathTaskCancel(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	uuid := fields.Get(fieldNameUUID).(string)
	return m.cancelTask(ctx, req.Storage, uuid)
}

func (m *Manager) cancelTask(ctx context.Context, reqStorage logical.Storage, uuid string) (*logical.Response, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Worker.HasRunningJobByTaskUUID(uuid) {
		// stop and run new worker
		m.Worker.Stop()
		m.startNewWorker()

		if err := markTaskAsCanceled(ctx, reqStorage, uuid); err != nil {
			return nil, err
		}

		if err := m.Storage.Delete(ctx, storageKeyCurrentRunningTask); err != nil {
			return nil, err
		}

		return nil, nil
	}

	return &logical.Response{
		Warnings: []string{
			fmt.Sprintf("task %q not running", uuid),
		},
	}, nil
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
			return nil, logical.ErrorResponse(fmt.Sprintf("task %q not found", uuid)), nil
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
