package tasks_manager

import (
	"context"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

type Interface interface {
	BackendInterface
	ActionsInterface
}

type BackendInterface interface {
	// Paths returns backend paths to work with tasks
	Paths() []*framework.Path

	// PeriodicTask performs a periodic task. Should be used as the PeriodicFunc of backend or be part of an existing implementation
	PeriodicTask(ctx context.Context, req *logical.Request) error
}

type ActionsInterface interface {
	// RunTask runs task or returns busy error
	RunTask(ctx context.Context, reqStorage logical.Storage, taskFunc func(ctx context.Context, storage logical.Storage) error) (string, error)

	// AddTask adds task to queue
	AddTask(ctx context.Context, reqStorage logical.Storage, taskFunc func(ctx context.Context, storage logical.Storage) error) (string, error)

	// AddOptionalTask adds task to queue if empty
	AddOptionalTask(ctx context.Context, reqStorage logical.Storage, taskFunc func(ctx context.Context, storage logical.Storage) error) (string, bool, error)
}
