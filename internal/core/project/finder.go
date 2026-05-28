package project

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/muhfaris/gherkio/internal/model"
	"github.com/muhfaris/gherkio/internal/runner"
)

// ProjectMeta holds absolute directories and info about the active Gherkio workspace.
type ProjectMeta struct {
	Name           string `json:"name"`
	Version        string `json:"version"`
	GherkioVersion string `json:"gherkioVersion"`
	RootDir        string `json:"rootDir"`
	TestsDir       string `json:"testsDir"`
	EnvsDir        string `json:"envsDir"`
	SchemasDir     string `json:"schemasDir"`
	ReportsDir     string `json:"reportsDir"`
}

// FindRoot walks up from cwd to find the directory containing the .gherkio folder.
func FindRoot(cwd string) (string, error) {
	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, ".gherkio")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no .gherkio directory found in any parent directory")
		}
		dir = parent
	}
}

// LoadConfig delegates config loading to the runner.
func LoadConfig(projectDir string) (*model.Config, error) {
	return runner.LoadConfig(projectDir)
}

// GetMeta reads the config and computes absolute directory paths.
func GetMeta(projectDir string) (*ProjectMeta, error) {
	cfg, err := LoadConfig(projectDir)
	if err != nil {
		return nil, err
	}

	meta := &ProjectMeta{
		Name:           cfg.Project.Name,
		Version:        cfg.Project.Version,
		GherkioVersion: cfg.GherkioVersion,
		RootDir:        projectDir,
	}

	// Resolve directories using config overrides or defaults
	testsPath := ".gherkio/tests"
	if cfg.Tests.Path != "" {
		testsPath = cfg.Tests.Path
	}
	meta.TestsDir = filepath.Join(projectDir, testsPath)

	envsPath := ".gherkio/environments"
	if cfg.Environments.Path != "" {
		envsPath = cfg.Environments.Path
	}
	meta.EnvsDir = filepath.Join(projectDir, envsPath)

	schemasPath := ".gherkio/schemas"
	if cfg.Schemas.Path != "" {
		schemasPath = cfg.Schemas.Path
	}
	meta.SchemasDir = filepath.Join(projectDir, schemasPath)

	reportsPath := ".gherkio/reports"
	if cfg.Reports.Path != "" {
		reportsPath = cfg.Reports.Path
	}
	meta.ReportsDir = filepath.Join(projectDir, reportsPath)

	if meta.Name == "" {
		meta.Name = filepath.Base(projectDir)
	}
	if meta.Version == "" {
		meta.Version = "0.1.0"
	}

	return meta, nil
}
