package publisher

import (
	"context"

	"github.com/hashicorp/vault/sdk/logical"
)

func (m *Publisher) PeriodicTask(_ context.Context, _ *logical.Request) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return nil
}
