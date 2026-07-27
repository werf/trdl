package repo

import "fmt"

type ChannelNotFoundLocallyError struct {
	RepoName string
	Group    string
	Channel  string
}

func NewChannelNotFoundLocallyError(repoName, group, channel string) error {
	return ChannelNotFoundLocallyError{
		RepoName: repoName,
		Group:    group,
		Channel:  channel,
	}
}

func (e ChannelNotFoundLocallyError) Error() string {
	return fmt.Sprintf("channel %[2]q not found locally (group: %[1]q)", e.Group, e.Channel)
}

type ChannelReleaseNotFoundLocallyError struct {
	RepoName string
	Release  string
	Group    string
	Channel  string
}

func NewChannelReleaseNotFoundLocallyError(repoName, group, channel, release string) error {
	return ChannelReleaseNotFoundLocallyError{
		RepoName: repoName,
		Release:  release,
		Group:    group,
		Channel:  channel,
	}
}

func (e ChannelReleaseNotFoundLocallyError) Error() string {
	return fmt.Sprintf("channel release %q not found locally (group: %q, channel: %q)", e.Release, e.Group, e.Channel)
}

type ChannelReleaseBinSeveralFilesFoundError struct {
	RepoName string
	Release  string
	Group    string
	Channel  string
	Names    []string
}

func NewChannelReleaseSeveralFilesFoundError(repoName, group, channel, release string, names []string) error {
	return ChannelReleaseBinSeveralFilesFoundError{
		RepoName: repoName,
		Release:  release,
		Group:    group,
		Channel:  channel,
		Names:    names,
	}
}

func (e ChannelReleaseBinSeveralFilesFoundError) Error() string {
	return fmt.Sprintf("several binary files found in release %q (group: %q, channel: %q)", e.Release, e.Group, e.Channel)
}

type ReleaseNotFoundLocallyError struct {
	RepoName string
	Version  string
}

func NewReleaseNotFoundLocallyError(repoName, version string) error {
	return ReleaseNotFoundLocallyError{
		RepoName: repoName,
		Version:  version,
	}
}

func (e ReleaseNotFoundLocallyError) Error() string {
	return fmt.Sprintf("version %q not found locally", e.Version)
}

type ReleaseBinSeveralFilesFoundError struct {
	RepoName string
	Version  string
	Names    []string
}

func NewReleaseBinSeveralFilesFoundError(repoName, version string, names []string) error {
	return ReleaseBinSeveralFilesFoundError{
		RepoName: repoName,
		Version:  version,
		Names:    names,
	}
}

func (e ReleaseBinSeveralFilesFoundError) Error() string {
	return fmt.Sprintf("several binary files found in version %q", e.Version)
}
