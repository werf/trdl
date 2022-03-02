package client

import "github.com/werf/trdl/client/pkg/repo"

type Interface interface {
	AddRepo(repoName, repoUrl string, rootVersion int64, rootSha512 string) error
	RemoveRepo(repoName string) error
	SetRepoDefaultChannel(repoName, channel string) error
	DoSelfUpdate(autocleanReleases bool) error
	UpdateRepoChannel(repoName, group, optionalChannel string, autocleanReleases bool) error
	UseRepoChannelReleaseBinDir(repoName, group, optionalChannel, shell string, opts repo.UseSourceOptions) (string, error)
	ExecRepoChannelReleaseBin(repoName, group, optionalChannel string, optionalBinName string, args []string) error
	GetRepoChannelReleaseDir(repoName, group, optionalChannel string) (string, error)
	GetRepoChannelReleaseBinDir(repoName, group, optionalChannel string) (string, error)
	GetRepoList() []*RepoConfiguration
	GetRepoClient(repoName string) (RepoInterface, error)
}

type RepoInterface interface {
	Setup(rootVersion int64, rootSha512 string) error
	UpdateChannel(group, channel string) error
	UseChannelReleaseBinDir(group, channel, shell string, opts repo.UseSourceOptions) (string, error)
	ExecChannelReleaseBin(group, channel string, optionalBinName string, args []string) error
	GetChannelRelease(group, channel string) (string, error)
	GetChannelReleaseDir(group, channel string) (string, error)
	GetChannelReleaseBinDir(group, channel string) (string, error)
	GetChannelReleaseBinPath(group, channel, optionalBinName string) (string, error)
	CleanReleases() error
}

type configurationInterface interface {
	RemoveRepoConfiguration(name string) error
	StageRepoConfiguration(name, url string)
	StageRepoDefaultChannel(name, channel string) error
	Reload() error
	Save(configPath string) error
	GetRepoConfiguration(name string) *RepoConfiguration
	GetRepoConfigurationList() []*RepoConfiguration
}
