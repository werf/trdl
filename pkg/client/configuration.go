package client

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"

	"github.com/werf/trdl/pkg/util"
)

const configurationFileBasename = "config.yaml"

type configuration struct {
	Projects []*ProjectConfiguration `yaml:"projects"`
}

func newConfiguration(configPath string) (configuration, error) {
	c := configuration{}

	if exist, err := util.IsRegularFileExist(configPath); err != nil {
		return c, fmt.Errorf("unable to check existence of file %q: %s", configPath, err)
	} else if !exist {
		return c, nil
	}

	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return c, fmt.Errorf("unable to read file %q: %s", configPath, err)
	}

	if err := yaml.Unmarshal(data, &c); err != nil {
		return c, fmt.Errorf("yaml unmarshalling failed: %s", err)
	}

	return c, nil
}

type ProjectConfiguration struct {
	Name    string
	RepoUrl string
}

func newProjectConfiguration(name, repoUrl string) *ProjectConfiguration {
	return &ProjectConfiguration{Name: name, RepoUrl: repoUrl}
}

func (c configuration) GetProjectConfigurations() []*ProjectConfiguration {
	return c.Projects
}

func (c configuration) GetProjectConfiguration(projectName string) *ProjectConfiguration {
	for i := range c.Projects {
		project := c.Projects[i]
		if project.Name == projectName {
			return project
		}
	}

	return nil
}

func (c *configuration) StageProjectConfiguration(projectName, repoUrl string) {
	project := c.GetProjectConfiguration(projectName)
	if project == nil {
		c.Projects = append(c.Projects, newProjectConfiguration(projectName, repoUrl))
		return
	}

	project.RepoUrl = repoUrl
}

func (c configuration) Save(configPath string) error {
	data, err := yaml.Marshal(&c)
	if err != nil {
		return fmt.Errorf("yaml marshalling failed: %s", err)
	}

	if err := ioutil.WriteFile(configPath, data, os.ModePerm); err != nil {
		return fmt.Errorf("unable to write file %q: %s", configPath, err)
	}

	return nil
}
