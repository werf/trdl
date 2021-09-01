package client

import "fmt"

type RepositoryNotInitializedErr struct {
	repoName string
}

func newRepositoryNotInitializedErr(repoName string) error {
	return &RepositoryNotInitializedErr{repoName: repoName}
}

func (e *RepositoryNotInitializedErr) Error() string {
	return fmt.Sprintf(
		"repository %q not initialized: configure it with \"trdl add\" command",
		e.repoName,
	)
}
