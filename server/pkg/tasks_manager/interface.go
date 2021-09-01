package tasks_manager

import (
	"context"

	"github.com/hashicorp/vault/sdk/logical"
)

type ActionsInterface interface {
	// RunTask runs task or returns busy error
	RunTask(ctx context.Context, reqStorage logical.Storage, taskFunc func(ctx context.Context, storage logical.Storage) error) (string, error)

	// AddTask adds task to queue
	AddTask(ctx context.Context, reqStorage logical.Storage, taskFunc func(ctx context.Context, storage logical.Storage) error) (string, error)

	// AddOptionalTask adds task to queue if empty
	AddOptionalTask(ctx context.Context, reqStorage logical.Storage, taskFunc func(ctx context.Context, storage logical.Storage) error) (string, bool, error)
}
