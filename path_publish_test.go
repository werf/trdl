package trdl

import (
	"testing"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/werf/vault-plugin-secrets-trdl/pkg/tasks_manager"
)

type PathPublishCallbackSuite struct {
	CommonSuite
}

func (suite *PathPublishCallbackSuite) SetupTest() {
	suite.CommonSuite.SetupTest()
	suite.req.Path = "publish"
	suite.req.Operation = logical.CreateOperation
}

func (suite *PathPublishCallbackSuite) TestConfigurationNotFound() {
	resp, err := suite.backend.HandleRequest(suite.ctx, suite.req)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), errorResponseConfigurationNotFound, resp)
}

func (suite *PathPublishCallbackSuite) TestBasic() {
	err := putConfiguration(suite.ctx, suite.storage, completeConfiguration())
	assert.Nil(suite.T(), err)

	suite.mockedPublisher.On("GetRepository").Return(nil)
	suite.mockedTasksManager.On("RunTask").Return("UUID", nil)

	resp, err := suite.backend.HandleRequest(suite.ctx, suite.req)
	assert.Nil(suite.T(), err)
	if assert.NotNil(suite.T(), resp) {
		assert.Equal(
			suite.T(),
			map[string]interface{}{
				"task_uuid": "UUID",
			},
			resp.Data,
		)
	}

	suite.mockedPublisher.AssertExpectations(suite.T())
	suite.mockedTasksManager.AssertExpectations(suite.T())
}

func (suite *PathPublishCallbackSuite) TestBusy() {
	err := putConfiguration(suite.ctx, suite.storage, completeConfiguration())
	assert.Nil(suite.T(), err)

	// tasks manager is busy
	suite.mockedTasksManager.IsBusy = true

	suite.mockedPublisher.On("GetRepository").Return(nil)
	suite.mockedTasksManager.On("RunTask").Return("", tasks_manager.ErrBusy)

	resp, err := suite.backend.HandleRequest(suite.ctx, suite.req)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), logical.ErrorResponse(tasks_manager.ErrBusy.Error()), resp)

	suite.mockedPublisher.AssertExpectations(suite.T())
	suite.mockedTasksManager.AssertExpectations(suite.T())
}

func TestBackendPathPublishCallback(t *testing.T) {
	suite.Run(t, new(PathPublishCallbackSuite))
}
