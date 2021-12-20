package config

import (
	"io/ioutil"

	"gopkg.in/yaml.v3"
)

type RawInputConfig struct {
	Type   string                 `yaml:"type"`
	Config map[string]interface{} `yaml:"config,inline"`
}

type CloggerConfig struct {
	Inputs []RawInputConfig `yaml:"inputs"`
}

func LoadConfigFile(path string) (*CloggerConfig, error) {
	body, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config := CloggerConfig{}
	err = yaml.Unmarshal(body, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
