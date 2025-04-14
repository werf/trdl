package server

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/werf/trdl/server/pkg/publisher"
	"github.com/werf/trdl/server/pkg/tasks_manager"
)

type BackendModuleInterface interface {
	Paths() []*framework.Path
	PeriodicFunc(ctx context.Context, req *logical.Request) error
}

type BackendPeriodicInterface interface {
	Periodic(ctx context.Context, req *logical.Request) error
}

type Backend struct {
	*framework.Backend
	TasksManager    tasks_manager.ActionsInterface
	Publisher       publisher.Interface
	BackendPeriodic BackendPeriodicInterface
}

var _ logical.Factory = Factory

func Factory(ctx context.Context, conf *logical.BackendConfig) (logical.Backend, error) {
	b, err := NewBackend(conf.Logger)
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

func NewBackend(logger hclog.Logger) (*Backend, error) {
	tasksManager := tasks_manager.NewManager(logger)
	publisher := publisher.NewPublisher(logger)

	b := &Backend{
		TasksManager: tasksManager,
		Publisher:    publisher,
	}
	b.BackendPeriodic = b

	b.Backend = &framework.Backend{
		BackendType: logical.TypeLogical,
		Help:        backendHelp,
	}

	b.InitPaths(tasksManager, publisher)
	b.InitPeriodicFunc(tasksManager, publisher)
	return b, nil
}

func (b *Backend) InitPaths(modules ...BackendModuleInterface) {
	b.Paths = framework.PathAppend(
		configurePaths(b),
		[]*framework.Path{
			releasePath(b),
			publishPath(b),
		},
	)

	for _, module := range modules {
		b.Paths = append(b.Paths, module.Paths()...)
	}
}

func (b *Backend) InitPeriodicFunc(modules ...BackendModuleInterface) {
	b.PeriodicFunc = func(ctx context.Context, request *logical.Request) error {
		for _, module := range modules {
			if err := module.PeriodicFunc(context.Background(), request); err != nil {
				return fmt.Errorf("backend module periodic task failed: %w", err)
			}
		}

		if b.BackendPeriodic != nil {
			if err := b.BackendPeriodic.Periodic(context.Background(), request); err != nil {
				return fmt.Errorf("backend main periodic task failed: %w", err)
			}
		}

		return nil
	}
}

const (
	backendHelp = `
The trdl plugin builds and releases software versions, publishes the release channels, and ensures object storage security via saving data signed by keys (no one has access to those keys) and continuously rotating TUF keys and metadata.
`
)
