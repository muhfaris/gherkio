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

// buildProjectStructureResource returns markdown describing the .gherkio/ directory layout.
func (s *Server) buildProjectStructureResource() string {
	return `# Gherkio Project Directory Structure

A Gherkio project lives under a .gherkio/ directory in the project root.
Here's what goes where:

## Tests (.gherkio/tests/)
Test scenario files in YAML format.

- **Create**: use create_test tool
- **Read**: use read_test or list_tests tool
- **Update**: use update_test tool
- **Delete**: use delete_test tool

## Credentials (.gherkio/credentials/)
Account credentials per environment.

Format:
- accounts: map of account name to credentials
  - username: account username
  - password: account password (auto-masked)
  - role: account role
  - Any extra fields are passed through

- **Create**: use create_credential tool
- **Read**: use read_credential tool or list_environments tool
- **Update**: use update_credential tool

## Environments (.gherkio/environments/)
Environment configuration with base URL and service overrides.

Format:
- baseUrl: base URL for all requests
- services: optional map of named service overrides
  - <name>: baseUrl for that service

- **Create**: use create_environment tool
- **Read**: use list_environments tool
- **Update**: use update_environment tool

## Schemas (.gherkio/schemas/)
Custom validation schemas for response body assertions.

- **Create**: use create_schema tool
- **Read**: use list_schemas tool
- **Update**: use update_schema tool

## Reports (.gherkio/reports/)
Auto-generated HTML/JSON reports (safe to .gitignore).
`
}
