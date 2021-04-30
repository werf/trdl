package tasks

import (
	"context"
	"fmt"
	"sync"

	"github.com/fatih/structs"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	fieldNameUUID   = "uuid"
	fieldNameLimit  = "limit"
	fieldNameOffset = "offset"
)

type Backend struct {
	*Manager
	mu sync.Mutex
}

func NewBackend() *Backend {
	return &Backend{Manager: newManager()}
}

func (b *Backend) PeriodicFunc(periodicTaskFunc func(ctx context.Context, storage logical.Storage) error) func(ctx context.Context, req *logical.Request) error {
	return func(_ context.Context, req *logical.Request) error {
		_ = b.RunOptionalTask(context.Background(), req.Storage, periodicTaskFunc)
		return nil
	}
}

func (b *Backend) Paths() []*framework.Path {
	return []*framework.Path{
		{
			Pattern: "task/?",
			Fields:  map[string]*framework.FieldSchema{},
			Operations: map[logical.Operation]framework.OperationHandler{
				logical.ReadOperation: &framework.PathOperation{
					Callback: b.pathTaskList,
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
					Callback: b.pathTaskStatus,
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
					Callback: b.pathTaskCancel,
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
					Callback: b.pathTaskLogRead,
				},
			},
		},
	}
}

func (b *Backend) pathTaskList(ctx context.Context, req *logical.Request, _ *framework.FieldData) (*logical.Response, error) {
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

func (b *Backend) pathTaskStatus(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
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

func (b *Backend) pathTaskCancel(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	uuid := fields.Get(fieldNameUUID).(string)
	if !b.Manager.HasRunningTask(uuid) {
		return &logical.Response{
			Warnings: []string{
				fmt.Sprintf("task %q not running", uuid),
			},
		}, nil
	}

	b.Manager.CancelRunningTask(uuid)
	if err := markTaskAsCanceled(ctx, req.Storage, uuid); err != nil {
		return nil, err
	}

	return nil, nil
}

func (b *Backend) pathTaskLogRead(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	offset := fields.Get(fieldNameOffset).(int)
	limit := fields.Get(fieldNameLimit).(int)
	uuid := fields.Get(fieldNameUUID).(string)

	data, resp, err := func() ([]byte, *logical.Response, error) {
		if b.Manager.HasRunningTask(uuid) {
			data := b.Manager.GetTaskLog(uuid)
			if data == nil {
				return nil, nil, nil
			}

			return data, nil, nil
		}

		data, err := getTaskLogFromStorage(ctx, req.Storage, uuid)
		if err != nil {
			return nil, nil, err
		}

		if data == nil {
			return nil, logical.ErrorResponse(fmt.Sprintf("task log %q not found", uuid)), nil
		}

		return data, nil, nil
	}()
	if err != nil {
		return nil, err
	}

	if resp != nil {
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

const uuidPatternRegexp = "(?i:[0-9A-F]{8}-[0-9A-F]{4}-[4][0-9A-F]{3}-[89AB][0-9A-F]{3}-[0-9A-F]{12})"

func uuidPattern(name string) string {
	return fmt.Sprintf(`(?P<%s>%s)`, name, uuidPatternRegexp)
}
