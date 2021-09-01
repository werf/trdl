package repo

import "fmt"

type ChannelNotFoundLocallyErr struct {
	RepoName string
	Group    string
	Channel  string
}

func NewChannelNotFoundLocallyErr(repoName, group, channel string) error {
	return ChannelNotFoundLocallyErr{
		RepoName: repoName,
		Group:    group,
		Channel:  channel,
	}
}

func (e ChannelNotFoundLocallyErr) Error() string {
	return fmt.Sprintf("channel %[2]q not found locally (group: %[1]q)", e.Group, e.Channel)
}

type ChannelReleaseNotFoundLocallyErr struct {
	RepoName string
	Release  string
	Group    string
	Channel  string
}

func NewChannelReleaseNotFoundLocallyErr(repoName, group, channel, release string) error {
	return ChannelReleaseNotFoundLocallyErr{
		RepoName: repoName,
		Release:  release,
		Group:    group,
		Channel:  channel,
	}
}

func (e ChannelReleaseNotFoundLocallyErr) Error() string {
	return fmt.Sprintf("channel release %q not found locally (group: %q, channel: %q)", e.Release, e.Group, e.Channel)
}

type ChannelReleaseBinSeveralFilesFoundErr struct {
	RepoName string
	Release  string
	Group    string
	Channel  string
	Names    []string
}

func NewChannelReleaseSeveralFilesFoundErr(repoName, group, channel, release string, names []string) error {
	return ChannelReleaseBinSeveralFilesFoundErr{
		RepoName: repoName,
		Release:  release,
		Group:    group,
		Channel:  channel,
		Names:    names,
	}
}

func (e ChannelReleaseBinSeveralFilesFoundErr) Error() string {
	return fmt.Sprintf("several binary files found in release %q (group: %q, channel: %q)", e.Release, e.Group, e.Channel)
}
