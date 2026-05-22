package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/muhfaris/gherkio/internal/runner"
)

// RenderJSON generates the JSON report string.
func RenderJSON(result *runner.RunResult, cfg ReportConfig, env string) (string, error) {
	// If --report-raw is set, we bypass masking (empty maskFields slice) for the data itself.
	// However, note that cURL generation always applies default masking internally if we don't pass fields.
	// We handle the raw logic by passing an empty slice to MapResultToReportData if MaskSensitive is false.
	var maskFields []string
	if cfg.MaskSensitive {
		maskFields = cfg.MaskFields
	} else {
		// Even for raw, we pass an empty slice to signify "no masking".
		maskFields = []string{}
	}

	data := MapResultToReportData(result, env, maskFields, !cfg.MaskSensitive)

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON report: %w", err)
	}

	return string(jsonData), nil
}

// SaveJSON saves the rendered JSON to the specified paths.
func SaveJSON(jsonData string, projectDir string, customPath string) (string, error) {
	basePath := filepath.Join(projectDir, ".gherkio", "reports")
	if customPath != "" {
		basePath = customPath
	}

	latestDir := filepath.Join(basePath, "latest")
	timestampDir := filepath.Join(basePath, time.Now().Format("20060102_150405"))

	if err := os.MkdirAll(latestDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create latest report dir: %w", err)
	}
	if err := os.MkdirAll(timestampDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create timestamp report dir: %w", err)
	}

	latestFile := filepath.Join(latestDir, "report.json")
	if err := os.WriteFile(latestFile, []byte(jsonData), 0644); err != nil {
		return "", fmt.Errorf("failed to write latest json report: %w", err)
	}

	timestampFile := filepath.Join(timestampDir, "report.json")
	if err := os.WriteFile(timestampFile, []byte(jsonData), 0644); err != nil {
		return "", fmt.Errorf("failed to write timestamp json report: %w", err)
	}

	return latestFile, nil
}
