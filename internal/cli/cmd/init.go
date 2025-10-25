package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize gherkio/ structure with examples",
	Long:  `Creates the necessary directories and sample files for gherkio to run.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInit(args)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(args []string) error {
	base := "gherkio"
	dirs := []string{
		filepath.Join(base, "envs"),
		filepath.Join(base, "apis"),
		filepath.Join(base, "flows"),
		filepath.Join(base, "fixtures"),
		filepath.Join(base, "features"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}
	// write sample files if not exist
	writeIfAbsent(filepath.Join(base, "envs", "dev.yaml"), sampleEnv())
	writeIfAbsent(filepath.Join(base, "apis", "core.yaml"), sampleAPIs())
	writeIfAbsent(filepath.Join(base, "flows", "auth.yaml"), sampleFlow())
	writeIfAbsent(filepath.Join(base, "fixtures", "users.create.json"), sampleFixture())
	writeIfAbsent(filepath.Join(base, "features", "sample.feature"), sampleFeature())
	fmt.Println("Initialized gherkio/ with examples. Happy testing! ✨")
	return nil
}

func writeIfAbsent(path, content string) {
	if _, err := os.Stat(path); err == nil {
		return
	}
	_ = os.WriteFile(path, []byte(content), 0o644)
}

func sampleEnv() string {
	return `version: 1
baseURL: https://jsonplaceholder.typicode.com
headers:
  X-App: gherkio
vars:
  # Seeds default credentials into the store so flows can reuse them.
  login.username: demo_user
  login.password: secret123
timeouts:
  request: 20s
retries:
  max: 1
  wait: 200ms
redact:
  headers: [Authorization, X-Api-Key]
  json: ["$.password", "$.token"]
`
}

func sampleAPIs() string {
	return `version: 1
auth:
  bearer:
    fromStore: access_token
    header: Authorization
    template: "Bearer {{ .value }}"
endpoints:
  users.getById:
    method: GET
    path: /users/{{ .id }}
    expect:
      status: 200
  users.create:
    method: POST
    path: /users
    headers:
      Content-Type: application/json
`
}

func sampleFlow() string {
	return `version: 1
flows:
  login:
    params: [username, password]
    steps:
      - call: auth.login
        body: |
          {"username":"{{ .username }}","password":"{{ .password }}"}
        save:
          "$.access_token": access_token
      - setAuth: bearer
`
}

func sampleFixture() string {
	return `{"name":"Ayu","email":"ayu@example.com"}
`
}

func sampleFeature() string {
	return `Feature: Sample journey

  Background:
    Given I load env "dev"
    And I load catalogs from "gherkio/apis"

  Scenario: Get user by id
    Given I set path params:
      | id | 1 |
    When I call API "users.getById"
    Then response status should be 200
`
}
