package trdl

import (
	"context"
	"fmt"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/tasks"
)

type backend struct {
	*framework.Backend
	TaskQueueBackend *tasks.Backend
}

var _ logical.Factory = Factory

func Factory(ctx context.Context, conf *logical.BackendConfig) (logical.Backend, error) {
	b, err := newBackend()
	if err != nil {
		return nil, err
	}

	if conf == nil {
		return nil, fmt.Errorf("configuration passed into backend is nil")
	}

	if err := b.Setup(ctx, conf); err != nil {
		return nil, err
	}

	return b, nil
}

func newBackend() (*backend, error) {
	b := &backend{
		TaskQueueBackend: tasks.NewBackend(),
	}

	b.Backend = &framework.Backend{
		BackendType: logical.TypeLogical,
		Help:        backendHelp,
		Paths: framework.PathAppend(
			[]*framework.Path{
				pathRelease(b),
			},
			b.TaskQueueBackend.Paths(),
		),
	}

	return b, nil
}

const (
	backendHelp = `
The TRDL backend plugin allows publishing of project's releases into the TUF compatible repository.
`
)
