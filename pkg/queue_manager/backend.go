package queue_manager

import (
	"context"
	"fmt"

	"github.com/fatih/structs"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	fieldNameUUID   = "uuid"
	fieldNameLimit  = "limit"
	fieldNameOffset = "offset"
)

func (m *Manager) Paths() []*framework.Path {
	return []*framework.Path{
		{
			Pattern: "task/?",
			Fields:  map[string]*framework.FieldSchema{},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: m.pathTaskList,
				},
			},
		},
		{
			Pattern: "task/" + uuidPattern(fieldNameUUID) + "$",
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
			Pattern: "task/" + uuidPattern(fieldNameUUID) + "/cancel$",
			Fields: map[string]*framework.FieldSchema{
				fieldNameUUID: {
					Type:     framework.TypeNameString,
					Required: true,
				},
			},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.UpdateOperation: &framework.PathOperation{
					Callback: m.pathTaskCancel,
				},
			},
		},
		{
			Pattern: "task/" + uuidPattern(fieldNameUUID) + "/logs$",
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
				logical.ReadOperation: &framework.PathOperation{
					Callback: m.pathTaskLogRead,
				},
			},
		},
	}
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

	if m.Queue == nil || !m.Queue.HasRunningTaskByUUID(uuid) {
		return &logical.Response{
			Warnings: []string{
				fmt.Sprintf("task %q not running", uuid),
			},
		}, nil
	}

	// cancel task queue
	{
		m.Queue.Stop()
		m.initQueue()
	}

	if err := markTaskAsCanceled(ctx, reqStorage, uuid); err != nil {
		return nil, err
	}

	return nil, nil
}

func (m *Manager) pathTaskLogRead(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	offset := fields.Get(fieldNameOffset).(int)
	limit := fields.Get(fieldNameLimit).(int)
	uuid := fields.Get(fieldNameUUID).(string)

	data, resp, err := m.readTaskLog(ctx, req.Storage, uuid)
	if err != nil {
		return nil, err
	} else if resp != nil {
		return resp, nil
	}

	if len(data) < offset {
		data = nil
	} else if len(data[offset:]) < limit {
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
	if m.Queue != nil && m.Queue.HasRunningTaskByUUID(uuid) {
		data := m.Queue.GetTaskLog(uuid)
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
