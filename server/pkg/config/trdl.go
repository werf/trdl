package config

import (
	"bytes"
	"errors"
	"fmt"
	"text/template"

	"gopkg.in/yaml.v2"

	"github.com/werf/trdl/server/pkg/docker"
)

const (
	DefaultTrdlPath = "trdl.yaml"
)

type Trdl struct {
	DockerImage    string   `yaml:"dockerImage,omitempty"`
	DockerImageOld string   `yaml:"docker_image,omitempty"` // legacy
	Commands       []string `yaml:"commands,omitempty"`
	BuilderImage   string   `yaml:"builderImage,omitempty"`
}

func (c *Trdl) GetDockerImage() string {
	if c.DockerImage != "" {
		return c.DockerImage
	}

	return c.DockerImageOld
}

func (c *Trdl) Validate() error {
	if c.GetDockerImage() == "" {
		return errors.New("\"dockerImage\" field must be set")
	} else if err := docker.ValidateImageNameWithDigest(c.GetDockerImage()); err != nil {
		return fmt.Errorf(`"dockerImage" field validation failed: %w'`, err)
	}

	if len(c.Commands) == 0 {
		return errors.New(`"commands" field must be set`)
	}

	if c.BuilderImage != "" {
		if err := docker.ValidateImageNameWithDigest(c.BuilderImage); err != nil {
			return fmt.Errorf(`"builderImage" field validation failed: %w'`, err)
		}
	}

	return nil
}

func ParseTrdl(data []byte, values map[string]interface{}) (*Trdl, error) {
	tmpl := template.New("trdl.yaml")
	if _, err := tmpl.Parse(string(data)); err != nil {
		return nil, fmt.Errorf("unable to parse template: %w", err)
	}

	buf := bytes.NewBuffer(nil)
	if err := tmpl.ExecuteTemplate(buf, "trdl.yaml", values); err != nil {
		return nil, fmt.Errorf("unable to execute template: %w", err)
	}

	var res *Trdl
	if err := yaml.Unmarshal(buf.Bytes(), &res); err != nil {
		return nil, fmt.Errorf("error unmarshalling yaml: %w", err)
	}

	return res, nil
}
