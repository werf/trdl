package server

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/werf/logboek"
	"github.com/werf/trdl/server/pkg/tasks_manager"
)

type capturingTasksManager struct {
	tasks_manager.ActionsInterface
	task func(ctx context.Context, storage logical.Storage) error
}

func (m *capturingTasksManager) RunTask(_ context.Context, _ logical.Storage, taskFunc func(ctx context.Context, storage logical.Storage) error) (string, error) {
	m.task = taskFunc
	return "UUID", nil
}

// TaskStorageSuite runs the release and publish tasks the way a builtin Vault
// plugin sees them: the router sets req.Storage to nil once the request handler
// returns, and the task starts later, on the storage handed to it by the tasks
// manager.
type TaskStorageSuite struct {
	CommonSuite
	tasksManager *capturingTasksManager
}

func (suite *TaskStorageSuite) SetupTest() {
	suite.CommonSuite.SetupTest()
	suite.tasksManager = &capturingTasksManager{}
	suite.backend.TasksManager = suite.tasksManager
	suite.mockedPublisher.On("GetRepository").Return(nil)
	suite.req.Operation = logical.CreateOperation
}

func (suite *TaskStorageSuite) TestReleaseTaskDoesNotUseRequestStorage() {
	trdlYaml := "dockerImage: alpine@sha256:0000000000000000000000000000000000000000000000000000000000000000\ncommands:\n  - true\n"
	cfg := completeConfiguration()
	cfg.GitRepoUrl = initGitRepository(suite.T(), map[string]string{"trdl.yaml": trdlYaml}, fieldGitTagValidValue)
	cfg.RequiredNumberOfVerifiedSignaturesOnCommit = 0
	cfg.BuildkitdAddress = "unix://" + filepath.Join(suite.T().TempDir(), "absent.sock")
	require.NoError(suite.T(), putConfiguration(suite.ctx, suite.storage, cfg))

	suite.req.Path = "release"
	suite.req.Data = map[string]interface{}{fieldNameGitTag: fieldGitTagValidValue}

	err := suite.runCapturedTask()
	require.ErrorContains(suite.T(), err, "can't build artifacts")
}

func (suite *TaskStorageSuite) TestPublishTaskDoesNotUseRequestStorage() {
	cfg := completeConfiguration()
	cfg.GitRepoUrl = initGitRepository(suite.T(), map[string]string{"trdl_channels.yaml": "{{{"}, "")
	cfg.InitialLastPublishedGitCommit = ""
	cfg.RequiredNumberOfVerifiedSignaturesOnCommit = 0
	require.NoError(suite.T(), putConfiguration(suite.ctx, suite.storage, cfg))

	suite.req.Path = "publish"

	err := suite.runCapturedTask()
	require.ErrorContains(suite.T(), err, "error getting trdl channels config")
}

func (suite *TaskStorageSuite) runCapturedTask() error {
	resp, err := suite.backend.HandleRequest(suite.ctx, suite.req)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), resp)
	require.Equal(suite.T(), "UUID", resp.Data["task_uuid"])
	require.NotNil(suite.T(), suite.tasksManager.task)

	suite.req.Storage = nil

	ctx, cancel := context.WithTimeout(logboek.NewContext(suite.ctx, logboek.DefaultLogger()), 2*time.Minute)
	defer cancel()

	var taskErr error
	require.NotPanics(suite.T(), func() {
		taskErr = suite.tasksManager.task(ctx, suite.storage)
	})
	return taskErr
}

func initGitRepository(t *testing.T, files map[string]string, tag string) string {
	dir := t.TempDir()
	repo, err := git.PlainInit(dir, false)
	require.NoError(t, err)
	worktree, err := repo.Worktree()
	require.NoError(t, err)

	for name, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
		_, err := worktree.Add(name)
		require.NoError(t, err)
	}

	signature := &object.Signature{Name: "trdl", Email: "trdl@example.com", When: time.Now()}
	commit, err := worktree.Commit("init", &git.CommitOptions{Author: signature})
	require.NoError(t, err)

	if tag != "" {
		_, err := repo.CreateTag(tag, commit, &git.CreateTagOptions{Tagger: signature, Message: tag})
		require.NoError(t, err)
	}

	return dir
}

func TestTaskStorage(t *testing.T) {
	suite.Run(t, new(TaskStorageSuite))
}
