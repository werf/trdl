package client

type Interface interface {
	AddProject(projectName, repoUrl string, rootVersion int64, rootSha512 string) error
	UpdateProjectChannel(projectName, group, channel string) error
	ProjectChannelReleasePath(projectName, group, channel string) (string, error)
	ListProjects() []*ProjectConfiguration
	ProjectClient(projectName string) (ProjectInterface, error)
}

type ProjectInterface interface {
	Init(rootVersion int64, rootSha512 string) error
	UpdateChannel(group, channel string) error
	ChannelReleasePath(group, channel string) (string, error)
}

type configurationInterface interface {
	GetProjectConfiguration(projectName string) *ProjectConfiguration
	GetProjectConfigurations() []*ProjectConfiguration
	StageProjectConfiguration(projectName, repoUrl string)
	Save(configPath string) error
}
