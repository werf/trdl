package trdl

import (
	"context"
	"sync"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

func Factory(ctx context.Context, c *logical.BackendConfig) (logical.Backend, error) {
	b := backend()
	if err := b.Setup(ctx, c); err != nil {
		return nil, err
	}
	return b, nil
}

type trdlBackend struct {
	*framework.Backend

	l sync.RWMutex

	providerCtx       context.Context
	providerCtxCancel context.CancelFunc
}

func backend() *trdlBackend {
	b := new(trdlBackend)
	b.providerCtx, b.providerCtxCancel = context.WithCancel(context.Background())

	b.Backend = &framework.Backend{
		BackendType: logical.TypeLogical,
		Help:        backendHelp,
		Paths: framework.PathAppend(
			[]*framework.Path{
				pathRelease(b),
			},
		),
	}

	return b
}

const (
	backendHelp = `
The TRDL backend plugin allows publishing of project's releases into the TUF compatible repository.
`
)
