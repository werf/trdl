package server

import (
	"testing"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type PathConfigureCallbacksSuite struct {
	CommonSuite
}

func (suite *PathConfigureCallbacksSuite) SetupTest() {
	suite.CommonSuite.SetupTest()
	suite.req.Path = "configure"
}

func (suite *PathConfigureCallbacksSuite) TestCreateOrUpdate_CompleteConfiguration() {
	reqData := dataCompleteConfiguration()

	suite.req.Operation = logical.CreateOperation
	suite.req.Data = reqData

	resp, err := suite.backend.HandleRequest(suite.ctx, suite.req)
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

			suite.req.Operation = logical.CreateOperation
			suite.req.Data = reqData

			resp, err := suite.backend.HandleRequest(suite.ctx, suite.req)
			assert.Nil(suite.T(), err)
			assert.Equal(suite.T(), logical.ErrorResponse("Required field %q must be set", requiredField), resp)
		})
	}
}

func (suite *PathConfigureCallbacksSuite) TestRead() {
	err := putConfiguration(suite.ctx, suite.storage, completeConfiguration())
	assert.Nil(suite.T(), err)

	suite.req.Operation = logical.ReadOperation

	resp, err := suite.backend.HandleRequest(suite.ctx, suite.req)
	assert.Nil(suite.T(), err)
	if assert.NotNil(suite.T(), resp) && assert.NotNil(suite.T(), resp.Data) {
		assert.Equal(suite.T(), dataCompleteConfiguration(), resp.Data)
	}
}

func (suite *PathConfigureCallbacksSuite) TestRead_ConfigurationNotFound() {
	suite.req.Operation = logical.ReadOperation

	resp, err := suite.backend.HandleRequest(suite.ctx, suite.req)
	assert.Nil(suite.T(), err)
	if assert.NotNil(suite.T(), resp) {
		assert.Equal(suite.T(), errorResponseConfigurationNotFound, resp)
	}
}

func (suite *PathConfigureCallbacksSuite) TestDelete() {
	err := putConfiguration(suite.ctx, suite.storage, completeConfiguration())
	assert.Nil(suite.T(), err)

	suite.req.Operation = logical.DeleteOperation

	resp, err := suite.backend.HandleRequest(suite.ctx, suite.req)
	assert.Nil(suite.T(), err)
	assert.Nil(suite.T(), resp)

	cfg, err := getConfiguration(suite.ctx, suite.storage)
	assert.Nil(suite.T(), err)
	assert.Nil(suite.T(), cfg)
}

func (suite *PathConfigureCallbacksSuite) TestDelete_ConfigurationNotFound() {
	suite.req.Operation = logical.DeleteOperation

	resp, err := suite.backend.HandleRequest(suite.ctx, suite.req)
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
		fieldNameGitTrdlPath:                                cfg.GitTrdlPath,
		fieldNameGitTrdlChannelsPath:                        cfg.GitTrdlChannelsPath,
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
		GitRepoUrl:                                 "https://github.com/werf/trdl/server.git",
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
