package client

type Interface interface {
	AddProject(projectName, repoUrl string, rootVersion int64, rootSha512 string) error
	UpdateProjectChannel(projectName, group, channel string) error
	ExecProjectChannelReleaseBin(projectName, group, channel string, optionalBinName string, args []string) error
	ProjectChannelReleaseDir(projectName, group, channel string) (string, error)
	ProjectChannelReleaseBinDir(projectName, group, channel string) (string, error)
	ListProjects() []*ProjectConfiguration
	ProjectClient(projectName string) (ProjectInterface, error)
}

type ProjectInterface interface {
	Init(repoUrl string, rootVersion int64, rootSha512 string) error
	UpdateChannel(group, channel string) error
	ExecChannelReleaseBin(group, channel string, optionalBinName string, args []string) error
	ChannelReleaseDir(group, channel string) (string, error)
	ChannelReleaseBinDir(group, channel string) (string, error)
}

type configurationInterface interface {
	GetProjectConfiguration(projectName string) *ProjectConfiguration
	GetProjectConfigurations() []*ProjectConfiguration
	StageProjectConfiguration(projectName, repoUrl string)
	Reload() error
	Save(configPath string) error
}
