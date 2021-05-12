package queue_manager

import (
	"context"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

type Interface interface {
	// Paths returns backend paths to work with tasks
	Paths() []*framework.Path

	// RunTask runs task or returns busy error
	RunTask(ctx context.Context, reqStorage logical.Storage, taskFunc func(ctx context.Context, storage logical.Storage) error) (string, error)

	// AddTask adds task to queue
	AddTask(ctx context.Context, reqStorage logical.Storage, taskFunc func(ctx context.Context, storage logical.Storage) error) (string, error)

	// AddOptionalTask adds task to queue if empty
	AddOptionalTask(ctx context.Context, reqStorage logical.Storage, taskFunc func(ctx context.Context, storage logical.Storage) error) (string, bool, error)
}
