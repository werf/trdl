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

func (suite *PathConfigureCallbacksSuite) TestAI_CreateOrUpdate_BuildxFieldsRejectedWithBuildkitdAddress() {
	for field, value := range map[string]interface{}{
		fieldNameBuildxDriver:     "kubernetes",
		fieldNameBuildxDriverOpts: []string{"namespace=trdl-build"},
	} {
		conflictingField := field
		suite.Run(conflictingField, func() {
			reqData := dataCompleteConfigurationWithoutBuildxFields()
			reqData[fieldNameBuildkitdAddress] = "tcp://buildkitd:1234"
			reqData[conflictingField] = value

			suite.req.Operation = logical.CreateOperation
			suite.req.Data = reqData

			resp, err := suite.backend.HandleRequest(suite.ctx, suite.req)
			assert.Nil(suite.T(), err)
			if assert.NotNil(suite.T(), resp) {
				assert.Contains(suite.T(), resp.Error().Error(), conflictingField)
				assert.Contains(suite.T(), resp.Error().Error(), fieldNameBuildkitdAddress)
			}

			cfg, err := getConfiguration(suite.ctx, suite.storage)
			assert.Nil(suite.T(), err)
			assert.Nil(suite.T(), cfg, "a rejected configuration must not be stored")
		})
	}
}

func (suite *PathConfigureCallbacksSuite) TestAI_CreateOrUpdate_OmittedBuildxFieldsWipeStoredOnes() {
	err := putConfiguration(suite.ctx, suite.storage, completeConfiguration())
	assert.Nil(suite.T(), err)

	// configure replaces the whole document, so omitting a field clears it: the
	// only way back to the buildx defaults is a write without these fields.
	reqData := dataCompleteConfigurationWithoutBuildxFields()

	suite.req.Operation = logical.UpdateOperation
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

func (suite *PathConfigureCallbacksSuite) TestAI_CreateOrUpdate_BlankValuesAreNotACombination() {
	// Resolution treats blank values as "not set", so the conflict check has to
	// agree with it instead of rejecting on the raw field value.
	for name, reqData := range map[string]map[string]interface{}{
		"blank address": func() map[string]interface{} {
			data := dataCompleteConfiguration()
			data[fieldNameBuildkitdAddress] = "   "

			return data
		}(),
		"blank driver opts": func() map[string]interface{} {
			data := dataCompleteConfigurationWithoutBuildxFields()
			data[fieldNameBuildkitdAddress] = "tcp://buildkitd:1234"
			data[fieldNameBuildxDriverOpts] = []string{"   "}

			return data
		}(),
	} {
		suite.Run(name, func() {
			suite.req.Operation = logical.CreateOperation
			suite.req.Data = reqData

			resp, err := suite.backend.HandleRequest(suite.ctx, suite.req)
			assert.Nil(suite.T(), err)
			assert.Nil(suite.T(), resp)
		})
	}
}

func (suite *PathConfigureCallbacksSuite) TestAI_CreateOrUpdate_RejectedCombinationKeepsConfiguration() {
	stored := completeConfiguration()
	err := putConfiguration(suite.ctx, suite.storage, stored)
	assert.Nil(suite.T(), err)

	reqData := dataCompleteConfigurationWithoutBuildxFields()
	reqData[fieldNameBuildkitdAddress] = "tcp://buildkitd:1234"
	reqData[fieldNameBuildxDriver] = "kubernetes"

	suite.req.Operation = logical.UpdateOperation
	suite.req.Data = reqData

	resp, err := suite.backend.HandleRequest(suite.ctx, suite.req)
	assert.Nil(suite.T(), err)
	assert.NotNil(suite.T(), resp)

	cfg, err := getConfiguration(suite.ctx, suite.storage)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), stored, cfg, "a rejected update must leave the stored configuration untouched")
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
