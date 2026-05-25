package mcp

import (
	"encoding/json"

	"github.com/muhfaris/gherkio/internal/runner"
)

// buildMatchersResource returns JSON describing all available assertion matchers.
// Fully dynamic — uses runner.GetMatchersInfo() as the single source of truth.
func (s *Server) buildMatchersResource() string {
	matcherInfo := runner.GetMatchersInfo()

	type outputInfo struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Usage       string `json:"usage"`
	}

	var out []outputInfo
	for _, m := range matcherInfo {
		usage := m.Name
		if m.HasArg {
			usage = m.Name + " <value>"
		}
		out = append(out, outputInfo{
			Name:        m.Name,
			Description: m.Description,
			Usage:       usage,
		})
	}

	data, _ := json.MarshalIndent(out, "", "  ")
	return string(data)
}

// buildVariablesResource returns JSON describing built-in generator variables.
// Fully dynamic — uses runner.GetVariableInfo() as the single source of truth.
func (s *Server) buildVariablesResource() string {
	vars := runner.GetVariableInfo()
	data, _ := json.MarshalIndent(vars, "", "  ")
	return string(data)
}

// buildPathsResource returns JSON describing canonical assertion paths.
// Fully dynamic — uses runner.GetPathInfo() as the single source of truth.
func (s *Server) buildPathsResource() string {
	paths := runner.GetPathInfo()
	data, _ := json.MarshalIndent(paths, "", "  ")
	return string(data)
}
