package config

import (
	"bytes"
	"errors"
	"fmt"
	"text/template"

	"gopkg.in/yaml.v2"

	"github.com/werf/vault-plugin-secrets-trdl/pkg/docker"
)

const (
	TrdlFileName = "trdl.yaml"
)

type Trdl struct {
	DockerImage string   `yaml:"docker_image,omitempty"`
	Commands    []string `yaml:"commands,omitempty"`
}

func (c *Trdl) Validate() error {
	if c.DockerImage == "" {
		return errors.New(`"docker_image" field must be set`)
	} else if err := docker.ValidateImageNameWithDigest(c.DockerImage); err != nil {
		return fmt.Errorf(`"docker_image" field validation failed: %s'`, err)
	}

	if len(c.Commands) == 0 {
		return errors.New(`"commands" field must be set`)
	}

	return nil
}

func ParseTrdl(data []byte, values map[string]interface{}) (*Trdl, error) {
	tmpl := template.New("trdl.yaml")
	if _, err := tmpl.Parse(string(data)); err != nil {
		return nil, fmt.Errorf("unable to parse template: %s", err)
	}

	buf := bytes.NewBuffer(nil)
	if err := tmpl.ExecuteTemplate(buf, "trdl.yaml", values); err != nil {
		return nil, fmt.Errorf("unable to execute template: %s", err)
	}

	var res *Trdl
	if err := yaml.Unmarshal(buf.Bytes(), &res); err != nil {
		return nil, fmt.Errorf("error unmarshalling yaml: %s", err)
	}

	return res, nil
}
