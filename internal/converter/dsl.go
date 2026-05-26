package converter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/muhfaris/gherkio/internal/model"
	"gopkg.in/yaml.v3"
)

type RequestFields struct {
	Service string            `yaml:"service,omitempty"`
	Method  string            `yaml:"method"`
	URL     string            `yaml:"url"`
	Headers map[string]string `yaml:"headers,omitempty"`
	Body    interface{}       `yaml:"body,omitempty"`
	Timeout string            `yaml:"timeout,omitempty"`
}

type ExpectFields struct {
	Status int `yaml:"status"`
}

type YamlStep struct {
	Request RequestFields `yaml:"request"`
	Expect  ExpectFields  `yaml:"expect"`
}

type YamlScenario struct {
	Scenario string     `yaml:"scenario"`
	Steps    []YamlStep `yaml:"steps"`
}

// ResolveURLForDSL checks if the parsed URL matches any environment or service base URL
// and strips it accordingly, returning the stripped path and the matching service name (if any).
func ResolveURLForDSL(parsedURL string, projectDir string, envName string) (string, string) {
	if projectDir == "" {
		return parsedURL, ""
	}
	if envName == "" {
		envName = "local"
	}

	envPath := filepath.Join(projectDir, ".gherkio", "environments", envName+".yaml")
	data, err := os.ReadFile(envPath)
	if err != nil {
		// Environment file not found, return full URL as-is
		return parsedURL, ""
	}

	var env model.Environment
	if err := yaml.Unmarshal(data, &env); err != nil {
		return parsedURL, ""
	}

	// 1. Check services first (longest base URL match wins)
	var bestService string
	var bestServiceBase string
	for name, svc := range env.Services {
		if svc.BaseURL != "" && strings.HasPrefix(parsedURL, svc.BaseURL) {
			if len(svc.BaseURL) > len(bestServiceBase) {
				bestService = name
				bestServiceBase = svc.BaseURL
			}
		}
	}

	if bestService != "" {
		stripped := strings.TrimPrefix(parsedURL, bestServiceBase)
		// Ensure leading slash
		if !strings.HasPrefix(stripped, "/") {
			stripped = "/" + stripped
		}
		return stripped, bestService
	}

	// 2. Check global BaseURL
	if env.BaseURL != "" && strings.HasPrefix(parsedURL, env.BaseURL) {
		stripped := strings.TrimPrefix(parsedURL, env.BaseURL)
		// Ensure leading slash
		if !strings.HasPrefix(stripped, "/") {
			stripped = "/" + stripped
		}
		return stripped, ""
	}

	return parsedURL, ""
}

// FormatYAML generates the Gherkio DSL representation of the request.
func FormatYAML(req *ParsedRequest, scenarioName string, stepOnly bool, projectDir string, envName string) (string, error) {
	strippedURL, serviceName := ResolveURLForDSL(req.URL, projectDir, envName)

	step := YamlStep{
		Request: RequestFields{
			Service: serviceName,
			Method:  req.Method,
			URL:     strippedURL,
			Headers: req.Headers,
			Body:    req.Body,
			Timeout: req.Timeout,
		},
		Expect: ExpectFields{
			Status: 200, // Safe default status assertion
		},
	}

	var node yaml.Node
	var err error

	if stepOnly {
		// Wrap in a slice so it marshals as:
		// - request:
		//     ...
		slice := []YamlStep{step}
		err = node.Encode(slice)
	} else {
		if scenarioName == "" {
			scenarioName = "untitled"
		}
		scen := YamlScenario{
			Scenario: scenarioName,
			Steps:    []YamlStep{step},
		}
		err = node.Encode(scen)
	}

	if err != nil {
		return "", fmt.Errorf("failed to encode YAML: %w", err)
	}

	// Set custom indent to 2 spaces (default is 4 for maps in yaml.v3 node marshalling)
	var buf strings.Builder
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&node); err != nil {
		return "", fmt.Errorf("failed to format YAML string: %w", err)
	}
	enc.Close()

	return buf.String(), nil
}
