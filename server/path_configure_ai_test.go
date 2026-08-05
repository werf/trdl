//go:build ai_tests

package server

import (
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
)

func (suite *PathConfigureCallbacksSuite) TestAI_CreateOrUpdate_BuildxFieldsOmitted() {
	reqData := dataCompleteConfiguration()
	delete(reqData, fieldNameBuildxDriver)
	delete(reqData, fieldNameBuildxDriverOpts)

	suite.req.Operation = logical.CreateOperation
	suite.req.Data = reqData

	resp, err := suite.backend.HandleRequest(suite.ctx, suite.req)
	assert.Nil(suite.T(), err)
	assert.Nil(suite.T(), resp)

	cfg, err := getConfiguration(suite.ctx, suite.storage)
	assert.Nil(suite.T(), err)
	if assert.NotNil(suite.T(), cfg) {
		assert.Empty(suite.T(), cfg.BuildxDriver)
		assert.Empty(suite.T(), cfg.BuildxDriverOpts)
	}
}

func (suite *PathConfigureCallbacksSuite) TestAI_CreateOrUpdate_UnsupportedBuildxDriver() {
	reqData := dataCompleteConfiguration()
	reqData[fieldNameBuildxDriver] = "docker"

	suite.req.Operation = logical.CreateOperation
	suite.req.Data = reqData

	resp, err := suite.backend.HandleRequest(suite.ctx, suite.req)
	assert.Nil(suite.T(), err)
	if assert.NotNil(suite.T(), resp) {
		assert.Contains(suite.T(), resp.Error().Error(), fieldNameBuildxDriver)
	}

	cfg, err := getConfiguration(suite.ctx, suite.storage)
	assert.Nil(suite.T(), err)
	assert.Nil(suite.T(), cfg)
}

func (suite *PathConfigureCallbacksSuite) TestAI_CreateOrUpdate_RejectedUpdateKeepsConfiguration() {
	err := putConfiguration(suite.ctx, suite.storage, completeConfiguration())
	assert.Nil(suite.T(), err)

	reqData := dataCompleteConfiguration()
	reqData[fieldNameBuildxDriver] = "docker"

	suite.req.Operation = logical.UpdateOperation
	suite.req.Data = reqData

	resp, err := suite.backend.HandleRequest(suite.ctx, suite.req)
	assert.Nil(suite.T(), err)
	assert.NotNil(suite.T(), resp)

	cfg, err := getConfiguration(suite.ctx, suite.storage)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), completeConfiguration(), cfg)
}

// A configuration written before the buildx fields existed carries neither key.
func (suite *PathConfigureCallbacksSuite) TestAI_Read_ConfigurationStoredBeforeBuildxFields() {
	entry := &logical.StorageEntry{
		Key: storageKeyConfiguration,
		Value: []byte(`{
			"git_repo_url": "https://github.com/werf/trdl/server.git",
			"required_number_of_verified_signatures_on_commit": 10,
			"s3_endpoint": "trdl.s3.us-west-2.example.com",
			"s3_region": "us-west-2",
			"s3_access_key_id": "AKIAIOSFODNN7EXAMPLE",
			"s3_secret_access_key": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			"s3_bucket_name": "trdl"
		}`),
	}
	assert.Nil(suite.T(), suite.storage.Put(suite.ctx, entry))

	cfg, err := getConfiguration(suite.ctx, suite.storage)
	assert.Nil(suite.T(), err)
	if assert.NotNil(suite.T(), cfg) {
		assert.Empty(suite.T(), cfg.BuildxDriver)
		assert.Empty(suite.T(), cfg.BuildxDriverOpts)
		assert.Equal(suite.T(), "trdl", cfg.S3BucketName)
	}
}
