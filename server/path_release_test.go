package server

import (
	"errors"
	"testing"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/werf/logboek"
	"github.com/werf/trdl/server/pkg/tasks_manager"
)

const fieldGitTagValidValue = "v1.0.1"

type PathReleaseCallbackSuite struct {
	CommonSuite
}

func (suite *PathReleaseCallbackSuite) SetupTest() {
	suite.CommonSuite.SetupTest()
	suite.req.Path = "release"
	suite.req.Operation = logical.CreateOperation
}

func (suite *PathReleaseCallbackSuite) TestRequiredGitTagField() {
	resp, err := suite.backend.HandleRequest(suite.ctx, suite.req)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), logical.ErrorResponse("Required field %q must be set", fieldNameGitTag), resp)
}

func (suite *PathReleaseCallbackSuite) TestConfigurationNotFound() {
	suite.req.Data = map[string]interface{}{fieldNameGitTag: fieldGitTagValidValue}

	resp, err := suite.backend.HandleRequest(suite.ctx, suite.req)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), errorResponseConfigurationNotFound, resp)
}

func (suite *PathReleaseCallbackSuite) TestBasic() {
	err := putConfiguration(suite.ctx, suite.storage, completeConfiguration())
	assert.Nil(suite.T(), err)

	suite.req.Data = map[string]interface{}{fieldNameGitTag: fieldGitTagValidValue}

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

func (suite *PathReleaseCallbackSuite) TestBusy() {
	err := putConfiguration(suite.ctx, suite.storage, completeConfiguration())
	assert.Nil(suite.T(), err)

	// tasks manager is busy
	suite.mockedTasksManager.IsBusy = true

	suite.req.Data = map[string]interface{}{fieldNameGitTag: fieldGitTagValidValue}

	suite.mockedPublisher.On("GetRepository").Return(nil)
	suite.mockedTasksManager.On("RunTask").Return("", tasks_manager.ErrBusy)

	resp, err := suite.backend.HandleRequest(suite.ctx, suite.req)
	assert.Nil(suite.T(), err)
	assert.Equal(suite.T(), logical.ErrorResponse("busy"), resp)

	suite.mockedPublisher.AssertExpectations(suite.T())
	suite.mockedTasksManager.AssertExpectations(suite.T())
}

func TestBackendPathReleaseCallback(t *testing.T) {
	suite.Run(t, new(PathReleaseCallbackSuite))
}

type PathReleaseTaskSuite struct {
	CommonSuite
	tasksManager *capturingTasksManager
}

func (suite *PathReleaseTaskSuite) SetupTest() {
	suite.CommonSuite.SetupTest()
	suite.tasksManager = &capturingTasksManager{}
	suite.backend.TasksManager = suite.tasksManager
	suite.mockedPublisher.On("GetRepository").Return(nil)
	suite.req.Path = "release"
	suite.req.Operation = logical.CreateOperation
	suite.req.Data = map[string]interface{}{fieldNameGitTag: fieldGitTagValidValue}
}

func (suite *PathReleaseTaskSuite) TestExistingReleaseIsRejected() {
	require.NoError(suite.T(), putConfiguration(suite.ctx, suite.storage, completeConfiguration()))
	suite.mockedPublisher.On("GetExistingReleases").Return([]string{"1.0.0", "1.0.1"}, nil)

	err := suite.runCapturedTask()
	require.EqualError(suite.T(), err, `release "1.0.1" already exists: an existing release cannot be overwritten, release a new version instead`)
	suite.mockedPublisher.AssertExpectations(suite.T())
}

func (suite *PathReleaseTaskSuite) TestNewReleaseProceeds() {
	cfg := completeConfiguration()
	cfg.GitRepoUrl = initGitRepository(suite.T(), map[string]string{"README.md": ""}, fieldGitTagValidValue)
	require.NoError(suite.T(), putConfiguration(suite.ctx, suite.storage, cfg))
	suite.mockedPublisher.On("GetExistingReleases").Return([]string{"1.0.0", "1.0.2"}, nil)

	err := suite.runCapturedTask()
	require.ErrorContains(suite.T(), err, "signature verification failed")
	suite.mockedPublisher.AssertExpectations(suite.T())
}

func (suite *PathReleaseTaskSuite) TestExistingReleasesError() {
	require.NoError(suite.T(), putConfiguration(suite.ctx, suite.storage, completeConfiguration()))
	suite.mockedPublisher.On("GetExistingReleases").Return([]string(nil), errors.New("s3 is unavailable"))

	err := suite.runCapturedTask()
	require.EqualError(suite.T(), err, "unable to get existing releases: s3 is unavailable")
	suite.mockedPublisher.AssertExpectations(suite.T())
}

func (suite *PathReleaseTaskSuite) runCapturedTask() error {
	resp, err := suite.backend.HandleRequest(suite.ctx, suite.req)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)
	require.NotNil(suite.T(), suite.tasksManager.task)

	return suite.tasksManager.task(logboek.NewContext(suite.ctx, logboek.DefaultLogger()), suite.storage)
}

func TestBackendPathReleaseTask(t *testing.T) {
	suite.Run(t, new(PathReleaseTaskSuite))
}
