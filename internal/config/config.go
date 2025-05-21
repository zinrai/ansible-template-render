package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Represents the configuration for a single playbook
type PlaybookConfig struct {
	Name      string `yaml:"name"`
	Inventory string `yaml:"inventory"`
}

// Represents runtime options for the tool
type Options struct {
	GenerateOnly bool   `yaml:"generate_only,omitempty"`
	LogLevel     string `yaml:"log_level,omitempty"`
}

// Represents the complete configuration for the tool
type Config struct {
	Playbooks []PlaybookConfig `yaml:"playbooks"`
	Options   Options          `yaml:"options,omitempty"`
}

// Loads the configuration from the specified path
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	// Validate the configuration
	if len(config.Playbooks) == 0 {
		return nil, fmt.Errorf("no playbooks specified in config")
	}

	for i, playbook := range config.Playbooks {
		if playbook.Name == "" {
			return nil, fmt.Errorf("playbook #%d missing name", i+1)
		}

		if playbook.Inventory == "" {
			return nil, fmt.Errorf("playbook '%s' missing inventory", playbook.Name)
		}
	}

	return &config, nil
}
