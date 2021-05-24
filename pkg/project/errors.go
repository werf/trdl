package project

import "fmt"

type ErrChannelNotFoundLocally struct {
	ProjectName string
	Group       string
	Channel     string
}

func NewErrChannelNotFoundLocally(projectName, group, channel string) error {
	return ErrChannelNotFoundLocally{
		ProjectName: projectName,
		Group:       group,
		Channel:     channel,
	}
}

func (e ErrChannelNotFoundLocally) Error() string {
	return fmt.Sprintf("channel not found locally (group: %q, channel: %q)", e.Group, e.Channel)
}

type ErrChannelReleaseNotFoundLocally struct {
	ProjectName string
	Release     string
	Group       string
	Channel     string
}

func NewErrChannelReleaseNotFoundLocally(projectName, group, channel, release string) error {
	return ErrChannelReleaseNotFoundLocally{
		ProjectName: projectName,
		Release:     release,
		Group:       group,
		Channel:     channel,
	}
}

func (e ErrChannelReleaseNotFoundLocally) Error() string {
	return fmt.Sprintf("channel release %q not found locally (group: %q, channel: %q)", e.Release, e.Group, e.Channel)
}

type ErrChannelReleaseBinSeveralFilesFound struct {
	ProjectName string
	Release     string
	Group       string
	Channel     string
	Names       []string
}

func NewErrChannelReleaseSeveralFilesFound(projectName, group, channel, release string, names []string) error {
	return ErrChannelReleaseBinSeveralFilesFound{
		ProjectName: projectName,
		Release:     release,
		Group:       group,
		Channel:     channel,
		Names:       names,
	}
}

func (e ErrChannelReleaseBinSeveralFilesFound) Error() string {
	return fmt.Sprintf("several binary files found in release %q (group: %q, channel: %q)", e.Release, e.Group, e.Channel)
}
