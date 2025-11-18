package setup

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the .gh-worktree.yml configuration
type Config struct {
	Setup SetupConfig `yaml:"setup"`
}

// SetupConfig contains post-creation setup commands
type SetupConfig struct {
	Run []string `yaml:"run"`
}

// LoadConfig loads the .gh-worktree.yml configuration from the main worktree
func LoadConfig(mainWorktreePath string) (*Config, error) {
	configPath := filepath.Join(mainWorktreePath, ".gh-worktree.yml")

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// No config file is not an error, just return empty config
		return &Config{}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}
