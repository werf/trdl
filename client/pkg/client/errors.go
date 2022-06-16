package client

import "fmt"

type RepositoryNotInitializedError struct {
	repoName string
}

func newRepositoryNotInitializedError(repoName string) error {
	return &RepositoryNotInitializedError{repoName: repoName}
}

func (e *RepositoryNotInitializedError) Error() string {
	return fmt.Sprintf(
		"repository %q not initialized: configure it with \"trdl add\" command",
		e.repoName,
	)
}
