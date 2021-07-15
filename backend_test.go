package trdl

import (
	"context"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/mock"

	"github.com/werf/vault-plugin-secrets-trdl/pkg/publisher"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/tasks_manager"
)

type MockedTasksManager struct {
	mock.Mock
	BackendModuleInterface
	tasks_manager.ActionsInterface
}

type MockedPublisher struct {
	mock.Mock
	BackendModuleInterface
	publisher.ActionsInterface
}

func (m *MockedPublisher) InitRepository(_ context.Context, _ logical.Storage, _ publisher.RepositoryOptions) error {
	m.Called()
	return nil
}
