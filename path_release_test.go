package trdl

import (
	"fmt"
	"testing"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/werf/vault-plugin-secrets-trdl/pkg/tasks_manager"
)

const fieldGitTagValidValue = "v1.0.1"

type PathReleaseCallbackSuite struct {
	CommonSuite
}

func (suite *PathReleaseCallbackSuite) TestRequiredGitTagField() {
	req := &logical.Request{
		Operation: logical.CreateOperation,
		Path:      "release",
		Storage:   suite.storage,
	}

	resp, err := suite.backend.HandleRequest(suite.ctx, req)
	assert.Nil(suite.T(), err)
	assert.Equal(
		suite.T(),
		logical.ErrorResponse(
			fmt.Sprintf("required field %q must be set", fieldNameGitTag),
		),
		resp,
	)
}

func (suite *PathReleaseCallbackSuite) TestConfigurationNotFound() {
	req := &logical.Request{
		Operation: logical.CreateOperation,
		Path:      "release",
		Data:      map[string]interface{}{"git_tag": fieldGitTagValidValue},
		Storage:   suite.storage,
	}

	resp, err := suite.backend.HandleRequest(suite.ctx, req)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), errorResponseConfigurationNotFound, resp)
}

func (suite *PathReleaseCallbackSuite) TestBasic() {
	err := putConfiguration(suite.ctx, suite.storage, completeConfiguration())
	assert.Nil(suite.T(), err)

	req := &logical.Request{
		Operation: logical.CreateOperation,
		Path:      "release",
		Data: map[string]interface{}{
			fieldNameGitTag: fieldGitTagValidValue,
		},
		Storage: suite.storage,
	}

	suite.mockedPublisher.On("GetRepository").Return(nil)
	suite.mockedTasksManager.On("RunTask").Return("UUID", nil)

	resp, err := suite.backend.HandleRequest(suite.ctx, req)
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

func (suite *PathReleaseCallbackSuite) TestBusy() {
	err := putConfiguration(suite.ctx, suite.storage, completeConfiguration())
	assert.Nil(suite.T(), err)

	// tasks manager is busy
	suite.mockedTasksManager.IsBusy = true

	req := &logical.Request{
		Operation: logical.CreateOperation,
		Path:      "release",
		Data: map[string]interface{}{
			fieldNameGitTag: fieldGitTagValidValue,
		},
		Storage: suite.storage,
	}

	suite.mockedPublisher.On("GetRepository").Return(nil)
	suite.mockedTasksManager.On("RunTask").Return("", tasks_manager.ErrBusy)

	resp, err := suite.backend.HandleRequest(suite.ctx, req)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), logical.ErrorResponse(tasks_manager.ErrBusy.Error()), resp)

	suite.mockedPublisher.AssertExpectations(suite.T())
	suite.mockedTasksManager.AssertExpectations(suite.T())
}

func TestBackendPathReleaseCallback(t *testing.T) {
	suite.Run(t, new(PathReleaseCallbackSuite))
}
