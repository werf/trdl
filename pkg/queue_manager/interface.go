package queue_manager

import (
	"context"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

type Interface interface {
	Paths() []*framework.Path
	RunOptionalTask(ctx context.Context, reqStorage logical.Storage, taskFunc func(ctx context.Context, storage logical.Storage) error) bool
	RunTask(ctx context.Context, reqStorage logical.Storage, taskFunc func(ctx context.Context, storage logical.Storage) error)
}
