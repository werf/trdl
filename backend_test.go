package trdl

import (
	"context"
	"testing"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/werf/vault-plugin-secrets-trdl/pkg/publisher"
	"github.com/werf/vault-plugin-secrets-trdl/pkg/tasks_manager"
)

var (
	pathMockedTasksManager = &framework.Path{Pattern: "tasks_manager"}
	pathMockedPublisher    = &framework.Path{Pattern: "publisher"}
)

type MockedTasksManager struct {
	mock.Mock
	tasks_manager.ActionsInterface
}

func (m *MockedTasksManager) Paths() []*framework.Path {
	m.Called()
	return []*framework.Path{pathMockedTasksManager}
}

func (m *MockedTasksManager) PeriodicFunc(_ context.Context, _ *logical.Request) error {
	m.Called()
	return nil
}

type MockedPublisher struct {
	mock.Mock
	publisher.ActionsInterface
}

func (m *MockedPublisher) Paths() []*framework.Path {
	m.Called()
	return []*framework.Path{pathMockedPublisher}
}

func (m *MockedPublisher) PeriodicFunc(_ context.Context, _ *logical.Request) error {
	m.Called()
	return nil
}

func (m *MockedPublisher) InitRepository(_ context.Context, _ logical.Storage, _ publisher.RepositoryOptions) error {
	m.Called()
	return nil
}

type CommonSuite struct {
	suite.Suite
	ctx                context.Context
	backend            *backend
	storage            logical.Storage
	mockedTasksManager *MockedTasksManager
	mockedPublisher    *MockedPublisher
}

func (suite *CommonSuite) SetupTest() {
	mockedTasksManager := &MockedTasksManager{}
	mockedPublisher := &MockedPublisher{}
	b := &backend{
		Backend:      &framework.Backend{},
		TasksManager: mockedTasksManager,
		Publisher:    mockedPublisher,
	}
	b.InitPaths()

	ctx := context.Background()
	storage := &logical.InmemStorage{}
	config := logical.TestBackendConfig()
	config.StorageView = storage
	err := b.Setup(ctx, config)
	assert.Nil(suite.T(), err)

	suite.ctx = ctx
	suite.backend = b
	suite.storage = storage
	suite.mockedTasksManager = mockedTasksManager
	suite.mockedPublisher = mockedPublisher
}

type BackendSuite struct {
	CommonSuite
}

func (suite *BackendSuite) TestInitPaths() {
	suite.mockedTasksManager.On("Paths").Return(nil)
	suite.mockedPublisher.On("Paths").Return(nil)

	suite.backend.InitPaths(suite.mockedTasksManager, suite.mockedPublisher)

	suite.mockedTasksManager.AssertExpectations(suite.T())
	suite.mockedPublisher.AssertExpectations(suite.T())

	assert.Contains(suite.T(), suite.backend.Paths, pathMockedTasksManager)
	assert.Contains(suite.T(), suite.backend.Paths, pathMockedPublisher)
}

func (suite *BackendSuite) TestInitPeriodicFunc() {
	suite.mockedTasksManager.On("PeriodicFunc").Return(nil)
	suite.mockedPublisher.On("PeriodicFunc").Return(nil)

	suite.backend.InitPeriodicFunc(suite.mockedTasksManager, suite.mockedPublisher)
	if assert.NotNil(suite.T(), suite.backend.PeriodicFunc) {
		_ = suite.backend.PeriodicFunc(context.Background(), nil)
	}

	suite.mockedTasksManager.AssertExpectations(suite.T())
	suite.mockedPublisher.AssertExpectations(suite.T())
}

func TestBackend(t *testing.T) {
	suite.Run(t, new(BackendSuite))
}
