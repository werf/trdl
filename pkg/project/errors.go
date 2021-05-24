package project

import "fmt"

func NewErrChannelNotFoundLocally(group, channel string) error {
	return ErrChannelNotFoundLocally{
		group:   group,
		channel: channel,
	}
}

type ErrChannelNotFoundLocally struct {
	group   string
	channel string
}

func (e ErrChannelNotFoundLocally) Error() string {
	return fmt.Sprintf("channel not found locally (group: %q, channel: %q)", e.group, e.channel)
}

func NewErrChannelReleaseNotFoundLocally(group, channel, release string) error {
	return ErrChannelReleaseNotFoundLocally{
		release: release,
		group:   group,
		channel: channel,
	}
}

type ErrChannelReleaseNotFoundLocally struct {
	release string
	group   string
	channel string
}

func (e ErrChannelReleaseNotFoundLocally) Error() string {
	return fmt.Sprintf("channel release %q not found locally (group: %q, channel: %q)", e.release, e.group, e.channel)
}
