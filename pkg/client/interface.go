package client

type Interface interface {
	AddProject(projectName, repoUrl string, rootVersion int64, rootSha512 string) error
	ListProjects() []*ProjectConfiguration
	ProjectClient(projectName string) (ProjectInterface, error)
}

type ProjectInterface interface {
	Init(rootVersion int64, rootSha512 string) error
}

type configurationInterface interface {
	GetProjectConfiguration(projectName string) *ProjectConfiguration
	GetProjectConfigurations() []*ProjectConfiguration
	StageProjectConfiguration(projectName, repoUrl string)
	Save(configPath string) error
}
