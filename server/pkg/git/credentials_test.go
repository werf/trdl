package git

import (
	"context"
	"testing"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type PathConfigureGitCredentialsCallbacksSuite struct {
	suite.Suite
	ctx     context.Context
	backend *framework.Backend
	storage logical.Storage
	req     *logical.Request
}

func (suite *PathConfigureGitCredentialsCallbacksSuite) SetupTest() {
	b := &framework.Backend{
		Paths: CredentialsPaths(),
	}

	ctx := context.Background()
	storage := &logical.InmemStorage{}
	config := logical.TestBackendConfig()
	config.StorageView = storage

	err := b.Setup(ctx, config)
	assert.Nil(suite.T(), err)

	req := &logical.Request{
		Path:    "configure/git_credential",
		Storage: storage,
		Data:    map[string]interface{}{},
	}

	suite.ctx = ctx
	suite.backend = b
	suite.storage = storage
	suite.req = req
}

func (suite *PathConfigureGitCredentialsCallbacksSuite) Test_CreateOrUpdate_NoPassword() {
	assert := assert.New(suite.T())

	suite.req.Operation = logical.CreateOperation
	suite.req.Data = map[string]interface{}{
		FieldNameGitCredentialUsername: "user",
	}

	resp, err := suite.backend.HandleRequest(suite.ctx, suite.req)
	assert.Nil(err)
	assert.Equal(logical.ErrorResponse("%q field value should not be empty", FieldNameGitCredentialPassword), resp)
}

func (suite *PathConfigureGitCredentialsCallbacksSuite) Test_CreateOrUpdate_NoUsername() {
	assert := assert.New(suite.T())

	suite.req.Operation = logical.CreateOperation
	suite.req.Data = map[string]interface{}{
		FieldNameGitCredentialPassword: "password",
	}

	resp, err := suite.backend.HandleRequest(suite.ctx, suite.req)
	assert.Nil(err)
	assert.Equal(logical.ErrorResponse("%q field value should not be empty", FieldNameGitCredentialUsername), resp)
}

func (suite *PathConfigureGitCredentialsCallbacksSuite) Test_CreateOrUpdate_FullValidConfig() {
	assert := assert.New(suite.T())

	suite.req.Operation = logical.CreateOperation
	suite.req.Data = map[string]interface{}{
		FieldNameGitCredentialUsername: "user",
		FieldNameGitCredentialPassword: "password",
	}

	resp, err := suite.backend.HandleRequest(suite.ctx, suite.req)
	assert.Nil(err)
	assert.Nil(resp)

	cfg, err := GetGitCredential(suite.ctx, suite.storage)
	assert.Nil(err)
	assert.Equal(
		&GitCredential{
			Username: "user",
			Password: "password",
		},
		cfg,
	)
}

func (suite *PathConfigureGitCredentialsCallbacksSuite) Test_Delete_NoConfig() {
	assert := assert.New(suite.T())

	suite.req.Operation = logical.DeleteOperation

	resp, err := suite.backend.HandleRequest(suite.ctx, suite.req)
	assert.Nil(err)
	assert.Nil(resp)
}

func (suite *PathConfigureGitCredentialsCallbacksSuite) Test_Delete_HasConfig() {
	assert := assert.New(suite.T())

	err := PutGitCredential(suite.ctx, suite.storage, GitCredential{
		Username: "user",
		Password: "password",
	})
	assert.Nil(err)

	suite.req.Operation = logical.DeleteOperation

	resp, err := suite.backend.HandleRequest(suite.ctx, suite.req)
	assert.Nil(err)
	assert.Nil(resp)

	cfg, err := GetGitCredential(suite.ctx, suite.storage)
	assert.Nil(err)
	assert.Nil(cfg)
}

func TestGitCredentials(t *testing.T) {
	suite.Run(t, new(PathConfigureGitCredentialsCallbacksSuite))
}
