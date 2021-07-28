package client

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/werf/trdl/pkg/util"
)

const configurationFileBasename = "config.yaml"

var repoConfigurationNotFoundErr = errors.New("configuration not found")

type configuration struct {
	Repositories []*RepoConfiguration `yaml:"repositories"`

	configPath string
}

func newConfiguration(configPath string) (configuration, error) {
	c := configuration{}
	c.configPath = configPath
	return c, c.load()
}

type RepoConfiguration struct {
	Name           string `yaml:"name"`
	Url            string `yaml:"url"`
	DefaultChannel string `yaml:"defaultChannel"`
}

func newRepoConfiguration(name, url string) *RepoConfiguration {
	return &RepoConfiguration{Name: name, Url: url}
}

func (c configuration) GetRepoConfigurationList() []*RepoConfiguration {
	return c.Repositories
}

func (c configuration) GetRepoConfiguration(name string) *RepoConfiguration {
	for i := range c.Repositories {
		repo := c.Repositories[i]
		if repo.Name == name {
			return repo
		}
	}

	return nil
}

func (c *configuration) StageRepoConfiguration(name, url string) {
	repo := c.GetRepoConfiguration(name)
	if repo == nil {
		c.Repositories = append(c.Repositories, newRepoConfiguration(name, url))
		return
	}

	repo.Url = url
}

func (c *configuration) StageRepoDefaultChannel(name, channel string) error {
	repo := c.GetRepoConfiguration(name)
	if repo == nil {
		return repoConfigurationNotFoundErr
	}

	repo.DefaultChannel = channel

	return nil
}

func (c *configuration) Reload() error {
	return c.load()
}

func (c *configuration) load() error {
	if exist, err := util.IsRegularFileExist(c.configPath); err != nil {
		return fmt.Errorf("unable to check existence of file %q: %s", c.configPath, err)
	} else if !exist {
		return nil
	}

	data, err := ioutil.ReadFile(c.configPath)
	if err != nil {
		return fmt.Errorf("unable to read file %q: %s", c.configPath, err)
	}

	if err := yaml.Unmarshal(data, &c); err != nil {
		return fmt.Errorf("yaml unmarshalling failed: %s", err)
	}

	return nil
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
