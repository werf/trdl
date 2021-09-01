package tasks_manager

import (
	"context"
	"fmt"
	"time"

	"github.com/fatih/structs"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/werf/trdl/server/pkg/tasks_manager/worker"
)

const (
	fieldNameTaskTimeout      = "task_timeout"
	fieldNameTaskHistoryLimit = "task_history_limit"
	fieldNameUUID             = "uuid"
	fieldNameLimit            = "limit"
	fieldNameOffset           = "offset"

	fieldDefaultTaskTimeout      = "30m"
	fieldDefaultTaskHistoryLimit = 10
	fieldDefaultLimit            = 500

	defaultTaskTimeoutDuration = 30 * time.Minute
)

var (
	pathPatternConfigure  = "task/configure/?"
	pathPatternTaskList   = "task/?"
	pathPatternTaskStatus = "task/" + uuidPattern(fieldNameUUID) + "$"
	pathPatternTaskCancel = "task/" + uuidPattern(fieldNameUUID) + "/cancel$"
	pathPatternTaskLog    = "task/" + uuidPattern(fieldNameUUID) + "/log$"

	errorResponseConfigurationNotFound = logical.ErrorResponse("Configuration not found")
)

func (m *Manager) Paths() []*framework.Path {
	return []*framework.Path{
		{
			Pattern: pathPatternConfigure,
			Fields: map[string]*framework.FieldSchema{
				fieldNameTaskTimeout: {
					Type:        framework.TypeDurationSecond,
					Description: "Task timeout",
					Default:     fieldDefaultTaskTimeout,
				},
				fieldNameTaskHistoryLimit: {
					Type:        framework.TypeInt,
					Description: "Task history limit",
					Default:     fieldDefaultTaskHistoryLimit,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Description: "Configure task manager",
					Callback:    m.pathConfigureCreateOrUpdate,
				},
				logical.UpdateOperation: &framework.PathOperation{
					Description: "Configure task manager",
					Callback:    m.pathConfigureCreateOrUpdate,
				},
				logical.ReadOperation: &framework.PathOperation{
					Description: "Get task manager configuration",
					Callback:    m.pathConfigureRead,
				},
			},
		},
		{
			Pattern: pathPatternTaskList,
			Fields:  map[string]*framework.FieldSchema{},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Description: "Get list of task UUIDs",
					Callback:    m.pathTaskList,
				},
			},
		},
		{
			Pattern: pathPatternTaskStatus,
			Fields: map[string]*framework.FieldSchema{
				fieldNameUUID: {
					Type:        framework.TypeNameString,
					Description: "Task UUID",
					Required:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Description: "Get task status",
					Callback:    m.pathTaskStatus,
				},
			},
		},
		{
			Pattern: pathPatternTaskCancel,
			Fields: map[string]*framework.FieldSchema{
				fieldNameUUID: {
					Type:        framework.TypeNameString,
					Description: "Task UUID",
					Required:    true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.CreateOperation: &framework.PathOperation{
					Description: "Cancel running task",
					Callback:    m.pathTaskCancel,
				},
				logical.UpdateOperation: &framework.PathOperation{
					Description: "Cancel running task",
					Callback:    m.pathTaskCancel,
				},
			},
		},
		{
			Pattern: pathPatternTaskLog,
			Fields: map[string]*framework.FieldSchema{
				fieldNameUUID: {
					Type:        framework.TypeNameString,
					Description: "Task UUID",
					Required:    true,
				},
				fieldNameLimit: {
					Type:        framework.TypeInt,
					Description: "Limit of characters",
					Default:     fieldDefaultLimit,
				},
				fieldNameOffset: {
					Type:        framework.TypeInt,
					Description: "Offset",
					Default:     0,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Description: "Get task log",
					Callback:    m.pathTaskLogRead,
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
		return nil, fmt.Errorf("unable to save configuration: %s", err)
	}

	return nil, nil
}

func (m *Manager) pathConfigureRead(ctx context.Context, req *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	c, err := getConfiguration(ctx, req.Storage)
	if err != nil {
		return nil, fmt.Errorf("unable to get configuration: %s", err)
	}

	if c == nil {
		return errorResponseConfigurationNotFound, nil
	}

	data := structs.Map(c)
	data[fieldNameTaskTimeout] = c.TaskTimeout / time.Second
	return &logical.Response{Data: data}, nil
}

func (m *Manager) pathTaskList(ctx context.Context, req *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
	var list []string
	for _, state := range []taskState{taskStateCompleted, taskStateRunning, taskStateQueued} {
		prefix := taskStorageKeyPrefix(state)
		l, err := req.Storage.List(ctx, prefix)
		if err != nil {
			return nil, fmt.Errorf("unable to list %q in storage: %s", prefix, err)
		}

		list = append(list, l...)
	}

	return logical.ListResponse(list), nil
}

func (m *Manager) pathTaskStatus(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	uuid := fields.Get(fieldNameUUID).(string)

	var task *Task
	for _, state := range []taskState{taskStateQueued, taskStateRunning, taskStateCompleted} {
		t, err := getTaskFromStorage(ctx, req.Storage, state, uuid)
		if err != nil {
			return nil, err
		}

		if t != nil {
			task = t
			break
		}
	}

	if task == nil {
		return logical.ErrorResponse("Task %q not found", uuid), nil
	}

	return &logical.Response{Data: structs.Map(task)}, nil
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
		return logical.ErrorResponse("Field %q cannot be negative", fieldNameOffset), nil
	}

	if limit < 0 {
		return logical.ErrorResponse("Field %q cannot be negative", fieldNameLimit), nil
	}

	data, resp, err := func() ([]byte, *logical.Response, error) {
		// try to get running task log
		{
			var data []byte
			hold := m.Worker.HoldRunningJobByTaskUUID(uuid, func(job *worker.Job) {
				data = job.Log()
			})

			if hold {
				return data, nil, nil
			}
		}

		// try to get completed task log
		{
			t, err := getTaskFromStorage(ctx, req.Storage, taskStateCompleted, uuid)
			if err != nil {
				return nil, nil, err
			}

			if t != nil {
				data, err := getTaskLogFromStorage(ctx, req.Storage, t.UUID)
				if err != nil {
					return nil, nil, fmt.Errorf("unable to get task log %q from storage: %s", uuid, err)
				}

				return data, nil, nil
			}
		}

		// check queued task
		{
			t, err := getTaskFromStorage(ctx, req.Storage, taskStateQueued, uuid)
			if err != nil {
				return nil, nil, err
			}

			if t != nil {
				return nil, logical.ErrorResponse("Task %q in queue", uuid), nil
			}
		}

		return nil, logical.ErrorResponse("Task %q not found", uuid), nil
	}()
	if err != nil {
		return nil, err
	} else if resp != nil {
		return resp, nil
	}

	switch {
	case len(data) <= offset:
		data = nil
	case len(data[offset:]) < limit || limit == 0:
		data = data[offset:]
	default:
		data = data[offset : offset+limit]
	}

	return &logical.Response{
		Data: map[string]interface{}{
			"result": string(data),
		},
	}, nil
}

const uuidPatternRegexp = "(?i:[0-9A-F]{8}-[0-9A-F]{4}-[4][0-9A-F]{3}-[89AB][0-9A-F]{3}-[0-9A-F]{12})"

func uuidPattern(name string) string {
	return fmt.Sprintf(`(?P<%s>%s)`, name, uuidPatternRegexp)
}
