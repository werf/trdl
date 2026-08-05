//go:build ai_tests

package server

import (
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
)

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
