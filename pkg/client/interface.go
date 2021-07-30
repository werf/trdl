package client

type Interface interface {
	AddRepo(repoName, repoUrl string, rootVersion int64, rootSha512 string) error
	SetRepoDefaultChannel(repoName, channel string) error
	UpdateRepoChannel(repoName, group, optionalChannel string) error
	UseRepoChannelReleaseBinDir(repoName, group, optionalChannel, shell string, asFile bool) error
	ExecRepoChannelReleaseBin(repoName, group, optionalChannel string, optionalBinName string, args []string) error
	GetRepoChannelReleaseDir(repoName, group, optionalChannel string) (string, error)
	GetRepoChannelReleaseBinDir(repoName, group, optionalChannel string) (string, error)
	GetRepoList() []*RepoConfiguration
	GetRepoClient(repoName string) (RepoInterface, error)
}

type RepoInterface interface {
	Setup(rootVersion int64, rootSha512 string) error
	UpdateChannel(group, channel string) error
	UseChannelReleaseBinDir(group, channel, shell string, asFile bool) error
	ExecChannelReleaseBin(group, channel string, optionalBinName string, args []string) error
	GetChannelReleaseDir(group, channel string) (string, error)
	GetChannelReleaseBinDir(group, channel string) (string, error)
}

type configurationInterface interface {
	StageRepoConfiguration(name, url string)
	StageRepoDefaultChannel(name, channel string) error
	Reload() error
	Save(configPath string) error
	GetRepoConfiguration(name string) *RepoConfiguration
	GetRepoConfigurationList() []*RepoConfiguration
}
