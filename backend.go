package trdl

import (
	"context"
	"fmt"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	"github.com/werf/vault-plugin-secrets-trdl/pkg/pgp"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/publisher"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/tasks_manager"
)

type BackendModuleInterface interface {
	Paths() []*framework.Path
	PeriodicFunc(ctx context.Context, req *logical.Request) error
}

type backend struct {
	*framework.Backend
	TasksManager tasks_manager.ActionsInterface
	Publisher    publisher.ActionsInterface
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
	tasksManager := tasks_manager.NewManager()
	publisherManager := publisher.NewPublisher()

	b := &backend{
		TasksManager: tasksManager,
		Publisher:    publisherManager,
	}

	b.Backend = &framework.Backend{
		BackendType: logical.TypeLogical,
		Help:        backendHelp,
	}

	b.InitPaths(tasksManager)
	b.InitPeriodicFunc(tasksManager, publisherManager)
	return b, nil
}

func (b *backend) InitPaths(modules ...BackendModuleInterface) {
	b.Paths = framework.PathAppend(
		[]*framework.Path{
			configurePath(b),
			configureGitCredentialPath(b),
			releasePath(b),
			publishPath(b),
		},
		pgp.Paths(),
	)

	for _, module := range modules {
		b.Paths = append(b.Paths, module.Paths()...)
	}
}

func (b *backend) InitPeriodicFunc(modules ...BackendModuleInterface) {
	b.PeriodicFunc = func(ctx context.Context, request *logical.Request) error {
		for _, module := range modules {
			if err := module.PeriodicFunc(context.Background(), request); err != nil {
				return err
			}
		}

		return nil
	}
}

const (
	backendHelp = `
The TRDL backend plugin allows publishing of project's releases into the TUF compatible repository.
`
)
