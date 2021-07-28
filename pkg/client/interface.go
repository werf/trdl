package client

type Interface interface {
	AddRepo(repoName, repoUrl string, rootVersion int64, rootSha512 string) error
	UpdateRepoChannel(repoName, group, channel string) error
	ExecRepoChannelReleaseBin(repoName, group, channel string, optionalBinName string, args []string) error
	GetRepoChannelReleaseDir(repoName, group, channel string) (string, error)
	GetRepoChannelReleaseBinDir(repoName, group, channel string) (string, error)
	GetRepoList() []*RepoConfiguration
	GetRepoClient(repoName string) (RepoInterface, error)
}

type RepoInterface interface {
	Init(repoUrl string, rootVersion int64, rootSha512 string) error
	UpdateChannel(group, channel string) error
	ExecChannelReleaseBin(group, channel string, optionalBinName string, args []string) error
	GetChannelReleaseDir(group, channel string) (string, error)
	GetChannelReleaseBinDir(group, channel string) (string, error)
}

type configurationInterface interface {
	StageRepoConfiguration(name, url string)
	Reload() error
	Save(configPath string) error
	GetRepoConfiguration(name string) *RepoConfiguration
	GetRepoConfigurationList() []*RepoConfiguration
}
