package runner

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/muhfaris/gherkio/internal/model"
	"gopkg.in/yaml.v3"
)

// LoadConfig reads the .gherkio/config.yaml file.
func LoadConfig(projectDir string) (*model.Config, error) {
	configPath := filepath.Join(projectDir, ".gherkio", "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default config if file doesn't exist
			return &model.Config{}, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg model.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}
