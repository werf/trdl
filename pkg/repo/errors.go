package repo

import "fmt"

type ErrChannelNotFoundLocally struct {
	RepoName string
	Group    string
	Channel  string
}

func NewErrChannelNotFoundLocally(repoName, group, channel string) error {
	return ErrChannelNotFoundLocally{
		RepoName: repoName,
		Group:    group,
		Channel:  channel,
	}
}

func (e ErrChannelNotFoundLocally) Error() string {
	return fmt.Sprintf("channel not found locally (group: %q, channel: %q)", e.Group, e.Channel)
}

type ErrChannelReleaseNotFoundLocally struct {
	RepoName string
	Release  string
	Group    string
	Channel  string
}

func NewErrChannelReleaseNotFoundLocally(repoName, group, channel, release string) error {
	return ErrChannelReleaseNotFoundLocally{
		RepoName: repoName,
		Release:  release,
		Group:    group,
		Channel:  channel,
	}
}

func (e ErrChannelReleaseNotFoundLocally) Error() string {
	return fmt.Sprintf("channel release %q not found locally (group: %q, channel: %q)", e.Release, e.Group, e.Channel)
}

type ErrChannelReleaseBinSeveralFilesFound struct {
	RepoName string
	Release  string
	Group    string
	Channel  string
	Names    []string
}

func NewErrChannelReleaseSeveralFilesFound(repoName, group, channel, release string, names []string) error {
	return ErrChannelReleaseBinSeveralFilesFound{
		RepoName: repoName,
		Release:  release,
		Group:    group,
		Channel:  channel,
		Names:    names,
	}
}

func (e ErrChannelReleaseBinSeveralFilesFound) Error() string {
	return fmt.Sprintf("several binary files found in release %q (group: %q, channel: %q)", e.Release, e.Group, e.Channel)
}
