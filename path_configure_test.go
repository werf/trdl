package trdl

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type PathConfigureCallbacksSuite struct {
	suite.Suite
	ctx                context.Context
	backend            *backend
	storage            logical.Storage
	mockedTasksManager *MockedTasksManager
	mockedPublisher    *MockedPublisher
}

func (suite *PathConfigureCallbacksSuite) SetupTest() {
	mockedTasksManager := &MockedTasksManager{}
	mockedPublisher := &MockedPublisher{}
	b := &backend{
		Backend:      &framework.Backend{},
		TasksManager: mockedTasksManager,
		Publisher:    mockedPublisher,
	}
	b.Paths = []*framework.Path{configurePath(b)}

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

func (suite *PathConfigureCallbacksSuite) TestCreateOrUpdate_CompleteConfiguration() {
	reqData := dataCompleteConfiguration()
	req := &logical.Request{
		Operation: logical.CreateOperation,
		Path:      "configure",
		Data:      reqData,
		Storage:   suite.storage,
	}

	suite.mockedPublisher.On("InitRepository").Return(nil)

	resp, err := suite.backend.HandleRequest(suite.ctx, req)
	assert.Nil(suite.T(), err)
	assert.Nil(suite.T(), resp)

	suite.mockedPublisher.AssertExpectations(suite.T())

	cfg, err := getConfiguration(suite.ctx, suite.storage)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), completeConfiguration(), cfg)
}

func (suite *PathConfigureCallbacksSuite) TestCreateOrUpdate_RequiredFields() {
	for _, field := range []string{
		fieldNameGitRepoUrl,
		fieldNameRequiredNumberOfVerifiedSignaturesOnCommit,
		fieldNameS3BucketName,
		fieldNameS3Endpoint,
		fieldNameS3Region,
		fieldNameS3AccessKeyID,
		fieldNameS3SecretAccessKey,
	} {
		requiredField := field
		suite.Run(requiredField, func() {
			reqData := dataCompleteConfiguration()
			delete(reqData, requiredField)
			req := &logical.Request{
				Operation: logical.CreateOperation,
				Path:      "configure",
				Data:      reqData,
				Storage:   suite.storage,
			}

			// check InitRepository was not called
			suite.mockedPublisher.AssertExpectations(suite.T())

			resp, err := suite.backend.HandleRequest(suite.ctx, req)
			assert.Nil(suite.T(), err)
			assert.Equal(
				suite.T(),
				logical.ErrorResponse(
					fmt.Sprintf("required field %q must be set", requiredField),
				),
				resp,
			)
		})
	}
}

func (suite *PathConfigureCallbacksSuite) TestRead() {
	err := putConfiguration(suite.ctx, suite.storage, completeConfiguration())
	assert.Nil(suite.T(), err)

	req := &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "configure",
		Storage:   suite.storage,
	}

	resp, err := suite.backend.HandleRequest(suite.ctx, req)
	assert.Nil(suite.T(), err)
	if assert.NotNil(suite.T(), resp) && assert.NotNil(suite.T(), resp.Data) {
		assert.Equal(suite.T(), dataCompleteConfiguration(), resp.Data)
	}
}

func (suite *PathConfigureCallbacksSuite) TestRead_ConfigurationNotFound() {
	req := &logical.Request{
		Operation: logical.ReadOperation,
		Path:      "configure",
		Storage:   suite.storage,
	}

	resp, err := suite.backend.HandleRequest(suite.ctx, req)
	assert.Nil(suite.T(), err)
	if assert.NotNil(suite.T(), resp) {
		assert.Equal(suite.T(), errorResponseConfigurationNotFound, resp)
	}
}

func (suite *PathConfigureCallbacksSuite) TestDelete() {
	err := putConfiguration(suite.ctx, suite.storage, completeConfiguration())
	assert.Nil(suite.T(), err)

	req := &logical.Request{
		Operation: logical.DeleteOperation,
		Path:      "configure",
		Storage:   suite.storage,
	}

	resp, err := suite.backend.HandleRequest(suite.ctx, req)
	assert.Nil(suite.T(), err)
	assert.Nil(suite.T(), resp)

	cfg, err := getConfiguration(suite.ctx, suite.storage)
	assert.Nil(suite.T(), err)
	assert.Nil(suite.T(), cfg)
}

func (suite *PathConfigureCallbacksSuite) TestDelete_ConfigurationNotFound() {
	req := &logical.Request{
		Operation: logical.DeleteOperation,
		Path:      "configure",
		Storage:   suite.storage,
	}

	resp, err := suite.backend.HandleRequest(suite.ctx, req)
	assert.Nil(suite.T(), err)
	assert.Nil(suite.T(), resp)
}

func TestBackendPathConfigureCallbacks(t *testing.T) {
	suite.Run(t, new(PathConfigureCallbacksSuite))
}

func dataCompleteConfiguration() map[string]interface{} {
	cfg := completeConfiguration()

	return map[string]interface{}{
		fieldNameGitRepoUrl:                                 cfg.GitRepoUrl,
		fieldNameGitTrdlChannelsBranch:                      cfg.GitTrdlChannelsBranch,
		fieldNameInitialLastPublishedGitCommit:              cfg.InitialLastPublishedGitCommit,
		fieldNameRequiredNumberOfVerifiedSignaturesOnCommit: cfg.RequiredNumberOfVerifiedSignaturesOnCommit,
		fieldNameS3Endpoint:                                 cfg.S3Endpoint,
		fieldNameS3Region:                                   cfg.S3Region,
		fieldNameS3AccessKeyID:                              cfg.S3AccessKeyID,
		fieldNameS3SecretAccessKey:                          cfg.S3SecretAccessKey,
		fieldNameS3BucketName:                               cfg.S3BucketName,
	}
}

func completeConfiguration() *configuration {
	return &configuration{
		GitRepoUrl:                                 "https://github.com/werf/vault-plugin-secrets-trdl.git",
		GitTrdlChannelsBranch:                      "master",
		InitialLastPublishedGitCommit:              "252da187d03e92369808718377f58b8333cf202a",
		RequiredNumberOfVerifiedSignaturesOnCommit: 10,
		S3Endpoint:                                 "trdl.s3.us-west-2.example.com",
		S3Region:                                   "us-west-2",
		S3AccessKeyID:                              "AKIAIOSFODNN7EXAMPLE",
		S3SecretAccessKey:                          "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		S3BucketName:                               "trdl",
	}
}
