package credentialstore

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/muhfaris/gherkio/internal/model"
	"gopkg.in/yaml.v3"
)

// CredInfo details discovered credential metadata.
type CredInfo struct {
	EnvName  string   `json:"envName"`
	Accounts []string `json:"accounts"`
	FilePath string   `json:"filePath"`
}

// List scans all credentials files under .gherkio/credentials/.
func List(projectDir string) ([]CredInfo, error) {
	credDir := filepath.Join(projectDir, ".gherkio", "credentials")
	var creds []CredInfo
	files, err := os.ReadDir(credDir)
	if err != nil {
		if os.IsNotExist(err) {
			return creds, nil
		}
		return nil, fmt.Errorf("failed to read credentials directory: %w", err)
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
		fullPath := filepath.Join(credDir, name)

		data, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}

		var cred model.Credentials
		if err := yaml.Unmarshal(data, &cred); err != nil {
			continue
		}

		creds = append(creds, CredInfo{
			EnvName:  envName,
			Accounts: cred.AccountNames(),
			FilePath: fullPath,
		})
	}

	return creds, nil
}

// Read loads and parses a single credentials file.
func Read(projectDir, envName string) (*model.Credentials, error) {
	fullPath := filepath.Join(projectDir, ".gherkio", "credentials", envName+".yaml")
	data, err := os.ReadFile(fullPath)
	if os.IsNotExist(err) {
		fullPath = filepath.Join(projectDir, ".gherkio", "credentials", envName+".yml")
		data, err = os.ReadFile(fullPath)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials for '%s': %w", envName, err)
	}

	var creds model.Credentials
	if err := yaml.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("failed to parse credentials '%s': %w", envName, err)
	}

	return &creds, nil
}

// Create writes a new credentials file for an environment.
func Create(projectDir, envName string, creds *model.Credentials) error {
	credDir := filepath.Join(projectDir, ".gherkio", "credentials")
	fullPath := filepath.Join(credDir, envName+".yaml")

	if _, err := os.Stat(fullPath); err == nil {
		return fmt.Errorf("credentials for '%s' already exist", envName)
	}

	if err := os.MkdirAll(credDir, 0755); err != nil {
		return fmt.Errorf("failed to create credentials directory: %w", err)
	}

	data, err := yaml.Marshal(creds)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	return os.WriteFile(fullPath, data, 0644)
}

// Update overwrites an existing credentials file.
func Update(projectDir, envName string, creds *model.Credentials) error {
	credDir := filepath.Join(projectDir, ".gherkio", "credentials")
	fullPath := filepath.Join(credDir, envName+".yaml")

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		fullPath = filepath.Join(credDir, envName+".yml")
		if _, errYml := os.Stat(fullPath); os.IsNotExist(errYml) {
			return fmt.Errorf("credentials for '%s' do not exist", envName)
		}
	}

	data, err := yaml.Marshal(creds)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	return os.WriteFile(fullPath, data, 0644)
}

// Delete removes a credentials file.
func Delete(projectDir, envName string) error {
	credDir := filepath.Join(projectDir, ".gherkio", "credentials")
	fullPath := filepath.Join(credDir, envName+".yaml")

	err := os.Remove(fullPath)
	if os.IsNotExist(err) {
		fullPath = filepath.Join(credDir, envName+".yml")
		err = os.Remove(fullPath)
	}

	if err != nil {
		return fmt.Errorf("failed to delete credentials for '%s': %w", envName, err)
	}

	return nil
}
