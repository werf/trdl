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

func (suite *PathConfigureCallbacksSuite) TestCreateOrUpdate_BuildkitdAddress() {
	reqData := dataCompleteConfigurationWithoutBuildxFields()
	reqData[fieldNameBuildkitdAddress] = "tcp://buildkitd:1234"

	suite.req.Operation = logical.CreateOperation
	suite.req.Data = reqData

	resp, err := suite.backend.HandleRequest(suite.ctx, suite.req)
	assert.Nil(suite.T(), err)
	assert.Nil(suite.T(), resp)

	cfg, err := getConfiguration(suite.ctx, suite.storage)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), "tcp://buildkitd:1234", cfg.BuildkitdAddress)
}

func (suite *PathConfigureCallbacksSuite) TestCreateOrUpdate_InvalidBuildkitdAddress() {
	reqData := dataCompleteConfigurationWithoutBuildxFields()
	reqData[fieldNameBuildkitdAddress] = "ssh://buildhost"

	suite.req.Operation = logical.CreateOperation
	suite.req.Data = reqData

	resp, err := suite.backend.HandleRequest(suite.ctx, suite.req)
	assert.Nil(suite.T(), err)
	if assert.NotNil(suite.T(), resp) {
		assert.Contains(suite.T(), resp.Error().Error(), fieldNameBuildkitdAddress)
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
		fieldNameBuildkitdAddress:                           cfg.BuildkitdAddress,
		fieldNameBuildxDriver:                               cfg.BuildxDriver,
		fieldNameBuildxDriverOpts:                           cfg.BuildxDriverOpts,
	}
}

// The buildx settings and buildkitd_address are mutually exclusive, so a request
// exercising the address has to drop the buildx pair the fixture carries.
func dataCompleteConfigurationWithoutBuildxFields() map[string]interface{} {
	reqData := dataCompleteConfiguration()
	delete(reqData, fieldNameBuildxDriver)
	delete(reqData, fieldNameBuildxDriverOpts)

	return reqData
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
		BuildxDriver:                               "kubernetes",
		BuildxDriverOpts:                           []string{"namespace=trdl-build", "nodeselector=disktype=ssd,zone=a"},
	}
}

func (suite *PathConfigureCallbacksSuite) TestCreateOrUpdate_BuildxFieldsOmitted() {
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

func (suite *PathConfigureCallbacksSuite) TestCreateOrUpdate_BuildxFieldsRejectedWithBuildkitdAddress() {
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

func (suite *PathConfigureCallbacksSuite) TestCreateOrUpdate_OmittedBuildxFieldsWipeStoredOnes() {
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

func (suite *PathConfigureCallbacksSuite) TestCreateOrUpdate_BlankValuesAreNotACombination() {
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

func (suite *PathConfigureCallbacksSuite) TestCreateOrUpdate_RejectedCombinationKeepsConfiguration() {
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

func (suite *PathConfigureCallbacksSuite) TestCreateOrUpdate_UnsupportedBuildxDriver() {
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

func (suite *PathConfigureCallbacksSuite) TestCreateOrUpdate_RejectedUpdateKeepsConfiguration() {
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
func (suite *PathConfigureCallbacksSuite) TestRead_ConfigurationStoredBeforeBuildxFields() {
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
