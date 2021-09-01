package config

import (
	"fmt"

	"gopkg.in/yaml.v2"
)

const (
	TrdlChannelsFileName = "trdl_channels.yaml"
)

type TrdlChannels struct {
	Groups []TrdlGroup `yaml:"groups,omitempty"`
}

type TrdlGroup struct {
	Name     string             `yaml:"name"`
	Channels []TrdlGroupChannel `yaml:"channels,omitempty"`
}

type TrdlGroupChannel struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

func ParseTrdlChannels(data []byte) (*TrdlChannels, error) {
	var res *TrdlChannels

	if err := yaml.Unmarshal(data, &res); err != nil {
		return nil, fmt.Errorf("error unmarshalling yaml: %s", err)
	}

	return res, nil
}
