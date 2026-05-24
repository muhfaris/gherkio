package envstore

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/muhfaris/gherkio/internal/core/project"
	"github.com/muhfaris/gherkio/internal/model"
	"gopkg.in/yaml.v3"
)

// EnvInfo details discovered environment metadata.
type EnvInfo struct {
	Name          string `json:"name"`
	BaseURL       string `json:"baseUrl"`
	ServicesCount int    `json:"servicesCount"`
}

// List scans all environment configs under .gherkio/environments/.
func List(projectDir string) ([]EnvInfo, error) {
	meta, err := project.GetMeta(projectDir)
	if err != nil {
		return nil, err
	}

	var envs []EnvInfo
	files, err := os.ReadDir(meta.EnvsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return envs, nil
		}
		return nil, fmt.Errorf("failed to read environments directory: %w", err)
	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}
		name := f.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}

		envName := strings.TrimSuffix(strings.TrimSuffix(name, ".yaml"), ".yml")
		env, err := Read(projectDir, envName)
		if err != nil {
			continue
		}

		envs = append(envs, EnvInfo{
			Name:          envName,
			BaseURL:       env.BaseURL,
			ServicesCount: len(env.Services),
		})
	}

	return envs, nil
}

// Read loads and parses a single environment file.
func Read(projectDir, name string) (*model.Environment, error) {
	meta, err := project.GetMeta(projectDir)
	if err != nil {
		return nil, err
	}

	fullPath := filepath.Join(meta.EnvsDir, name+".yaml")
	data, err := os.ReadFile(fullPath)
	if os.IsNotExist(err) {
		// Try .yml fallback
		fullPath = filepath.Join(meta.EnvsDir, name+".yml")
		data, err = os.ReadFile(fullPath)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to read environment '%s': %w", name, err)
	}

	var env model.Environment
	if err := yaml.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("failed to parse environment '%s': %w", name, err)
	}

	return &env, nil
}

// Create writes a new environment config file.
func Create(projectDir, name string, env *model.Environment) error {
	meta, err := project.GetMeta(projectDir)
	if err != nil {
		return err
	}

	fullPath := filepath.Join(meta.EnvsDir, name+".yaml")
	if _, err := os.Stat(fullPath); err == nil {
		return fmt.Errorf("environment '%s' already exists", name)
	}

	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	data, err := yaml.Marshal(env)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	return os.WriteFile(fullPath, data, 0644)
}

// Update overwrites an existing environment config file.
func Update(projectDir, name string, env *model.Environment) error {
	meta, err := project.GetMeta(projectDir)
	if err != nil {
		return err
	}

	fullPath := filepath.Join(meta.EnvsDir, name+".yaml")
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		// Try .yml
		fullPath = filepath.Join(meta.EnvsDir, name+".yml")
		if _, errYml := os.Stat(fullPath); os.IsNotExist(errYml) {
			return fmt.Errorf("environment '%s' does not exist", name)
		}
	}

	data, err := yaml.Marshal(env)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	return os.WriteFile(fullPath, data, 0644)
}

// Delete removes an environment file, preventing deletion of the configured default environment.
func Delete(projectDir, name string) error {
	cfg, err := project.LoadConfig(projectDir)
	if err == nil && cfg.Environments.Default == name {
		return fmt.Errorf("cannot delete default environment '%s' configured in config.yaml", name)
	}

	meta, err := project.GetMeta(projectDir)
	if err != nil {
		return err
	}

	fullPath := filepath.Join(meta.EnvsDir, name+".yaml")
	err = os.Remove(fullPath)
	if os.IsNotExist(err) {
		fullPath = filepath.Join(meta.EnvsDir, name+".yml")
		err = os.Remove(fullPath)
	}

	if err != nil {
		return fmt.Errorf("failed to delete environment '%s': %w", name, err)
	}

	return nil
}
