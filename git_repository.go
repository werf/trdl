package trdl

import (
	"github.com/go-git/go-git/v5"
	trdlGit "github.com/werf/vault-plugin-secrets-trdl/pkg/git"
)

func cloneGitRepositoryBranch(url string, gitBranch string) (*git.Repository, error) {
	cloneGitOptions := trdlGit.CloneOptions{
		BranchName:        gitBranch,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
	}

	gitRepo, err := trdlGit.CloneInMemory(url, cloneGitOptions)
	if err != nil {
		return nil, err
	}

	return gitRepo, nil
}

func cloneGitRepositoryTag(url string, gitTag string) (*git.Repository, error) {
	cloneGitOptions := trdlGit.CloneOptions{
		TagName:           gitTag,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
	}

	gitRepo, err := trdlGit.CloneInMemory(url, cloneGitOptions)
	if err != nil {
		return nil, err
	}

	return gitRepo, nil
}
