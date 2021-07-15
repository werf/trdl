package publisher

import (
	"context"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

func (m *Publisher) Paths() []*framework.Path {
	return nil
}

func (m *Publisher) PeriodicFunc(_ context.Context, _ *logical.Request) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return nil
}
