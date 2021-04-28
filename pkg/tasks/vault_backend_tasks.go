package tasks

import (
	"context"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

type VaultBackendTasks struct {
	TasksQueue *Queue // Probably there will be multiple tasks queues, for different types of actions (/release or /release_channels for example), but there should be is a single /task* API path for all of them
	Storage    logical.Storage
}

func NewVaultBackendTasks(ctx context.Context, storage logical.Storage) *VaultBackendTasks {
	return &VaultBackendTasks{
		TasksQueue: NewQueue(ctx),
		// Storage is available only in periodic — think how to resolve this initialization casus
		// Probably just reinitialize it in the periodic func
	}
}

func (tasks *VaultBackendTasks) RunScheduledTask(runner func(ctx context.Context) error) string {
	return tasks.TasksQueue.RunScheduledTask(runner /*, Important: set RunScheduledTask additional on finish callback which should save data into the storage  */)
}

func (tasks *VaultBackendTasks) RunQueuedTask(runner func(ctx context.Context) error) {
	tasks.TasksQueue.RunScheduledTask(runner /*, Important: set RunScheduledTask additional on finish callback which should save data into the storage  */)
}

func (tasks *VaultBackendTasks) GetBackendPaths() []*framework.Path {
	var paths []*framework.Path

	paths = append(paths, &framework.Path{
		Pattern: "task$",
		Fields:  map[string]*framework.FieldSchema{},
		Operations: map[logical.Operation]framework.OperationHandler{
			logical.UpdateOperation: &framework.PathOperation{
				Callback: tasks.PathList,
			},
		},
	})

	return paths
}

func (tasks *VaultBackendTasks) PathList(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	// Get active tasks from the TasksQueue + tasks from storage

	return nil, nil
}

func (tasks *VaultBackendTasks) PathStatus(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	// Get task_id param from request
	// Get active task status from the TasksQueue or old task status from the storage

	return nil, nil
}

func (tasks *VaultBackendTasks) PathCancel(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	// Get task_id param from request
	// Cancel active task in the TasksQueue

	return nil, nil
}

func (tasks *VaultBackendTasks) PathLogs(ctx context.Context, req *logical.Request, fields *framework.FieldData) (*logical.Response, error) {
	// Get task_id param from request
	// Get active task logs from the TasksQueue or old task logs from the storage

	return nil, nil
}
