package client

import "github.com/werf/trdl/client/pkg/repo"

type Interface interface {
	AddRepo(repoName, repoUrl string, rootVersion int64, rootSha512 string) error
	RemoveRepo(repoName string) error
	SetRepoDefaultChannel(repoName, channel string) error
	DoSelfUpdate(autocleanReleases bool) error
	UpdateRepoChannel(repoName, group, optionalChannel string, autocleanReleases bool) error
	UseRepoChannelReleaseBinDir(repoName, group, optionalChannel, shell string, opts repo.UseSourceOptions) (string, error)
	ExecRepoChannelReleaseBin(repoName, group, optionalChannel, optionalBinName string, args []string) error
	GetRepoChannelReleaseDir(repoName, group, optionalChannel string) (string, error)
	GetRepoChannelReleaseBinDir(repoName, group, optionalChannel string) (string, error)
	UpdateRepoToVersion(repoName, version string, autocleanReleases bool) error
	UseRepoReleaseBinDir(repoName, version, shell string, opts repo.UseSourceOptions) (string, error)
	ExecRepoReleaseBin(repoName, version, optionalBinName string, args []string) error
	GetRepoReleaseBinDir(repoName, version string) (string, error)
	GetRepoReleaseDir(repoName, version string) (string, error)
	GetRepoList() []*RepoConfiguration
	GetRepoClient(repoName string) (RepoInterface, error)
}

type RepoInterface interface {
	Setup(rootVersion int64, rootSha512 string) error
	UpdateChannel(group, channel string) error
	UseChannelReleaseBinDir(group, channel, shell string, opts repo.UseSourceOptions) (string, error)
	ExecChannelReleaseBin(group, channel, optionalBinName string, args []string) error
	GetChannelRelease(group, channel string) (string, error)
	GetChannelReleaseDir(group, channel string) (string, error)
	GetChannelReleaseBinDir(group, channel string) (string, error)
	GetChannelReleaseBinPath(group, channel, optionalBinName string) (string, error)
	UpdateToVersion(version string) error
	UseReleaseBinDir(version, shell string, opts repo.UseSourceOptions) (string, error)
	ExecReleaseBin(version, optionalBinName string, args []string) error
	GetReleaseDir(version string) (string, error)
	GetReleaseBinDir(version string) (string, error)
	GetReleaseBinPath(version, optionalBinName string) (string, error)
	CleanReleases() error
	FindLocalReleaseByVersion(version string) (string, error)
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
