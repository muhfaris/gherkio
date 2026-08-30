package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindRootAndGetMeta(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Create a dummy .gherkio project structure
	gDir := filepath.Join(tmpDir, ".gherkio")
	err := os.MkdirAll(filepath.Join(gDir, "tests"), 0755)
	if err != nil {
		t.Fatalf("failed to setup: %v", err)
	}

	configYAML := `
project:
  name: "test-mcp-project"
  version: "2.3.4"
`
	err = os.WriteFile(filepath.Join(gDir, "config.yaml"), []byte(configYAML), 0644)
	if err != nil {
		t.Fatalf("failed to setup config: %v", err)
	}

	// 2. Test FindRoot from a nested directory
	nestedDir := filepath.Join(tmpDir, "nested", "deeply")
	_ = os.MkdirAll(nestedDir, 0755)

	root, err := FindRoot(nestedDir)
	if err != nil {
		t.Fatalf("FindRoot failed: %v", err)
	}
	if root != tmpDir {
		t.Errorf("expected root %s, got %s", tmpDir, root)
	}

	// 3. Test GetMeta
	meta, err := GetMeta(tmpDir)
	if err != nil {
		t.Fatalf("GetMeta failed: %v", err)
	}

	if meta.Name != "test-mcp-project" {
		t.Errorf("expected name test-mcp-project, got %s", meta.Name)
	}
	if meta.Version != "2.3.4" {
		t.Errorf("expected version 2.3.4, got %s", meta.Version)
	}
	if meta.TestsDir != filepath.Join(tmpDir, ".gherkio/tests") {
		t.Errorf("unexpected tests dir: %s", meta.TestsDir)
	}
}

func TestValidateProject(t *testing.T) {
	tmpDir := t.TempDir()

	// Setup a Gherkio project manually
	gDir := filepath.Join(tmpDir, ".gherkio")
	testsDir := filepath.Join(gDir, "tests")
	schemasDir := filepath.Join(gDir, "schemas")
	_ = os.MkdirAll(testsDir, 0755)
	_ = os.MkdirAll(schemasDir, 0755)
	_ = os.WriteFile(filepath.Join(gDir, "config.yaml"), []byte("environments:\n  default: local\n"), 0644)
	_ = os.MkdirAll(filepath.Join(gDir, "credentials"), 0755)
	_ = os.WriteFile(filepath.Join(gDir, "credentials", "local.yaml"), []byte("accounts:\n  user:\n    token: \"foo\"\n"), 0644)

	// 1. Positive case: fully valid test file
	validYaml := `scenario: Valid Login
steps:
  - request:
      method: POST
      url: https://api.example.com/login
    save:
      token: body.token
  - request:
      method: GET
      url: https://api.example.com/profile
      headers:
        Authorization: Bearer $token
`
	_ = os.WriteFile(filepath.Join(testsDir, "valid.yaml"), []byte(validYaml), 0644)

	stepPrefixedYaml := `scenario: Valid Step-Prefixed Variable
steps:
  - request:
      method: POST
      url: https://api.example.com/tickets
    save:
      1-ticketId: body.id
  - request:
      method: GET
      url: https://api.example.com/tickets/${1-ticketId}
`
	_ = os.WriteFile(filepath.Join(testsDir, "step_prefixed.yaml"), []byte(stepPrefixedYaml), 0644)

	// 2. Negative case: undefined variable
	undefVarYaml := `scenario: Undefined Var
steps:
  - request:
      method: GET
      url: https://api.example.com/profile
      headers:
        Authorization: Bearer $nonexistent_token
`
	_ = os.WriteFile(filepath.Join(testsDir, "undefined_var.yaml"), []byte(undefVarYaml), 0644)

	// 3. Negative case: undefined account
	undefAccountYaml := `scenario: Undefined Account
steps:
  - request:
      method: GET
      url: https://api.example.com/profile
      headers:
        Authorization: Bearer $accounts.admin.token
`
	_ = os.WriteFile(filepath.Join(testsDir, "undefined_account.yaml"), []byte(undefAccountYaml), 0644)

	// 4. Edge case: non-existent file validation directly
	t.Run("non-existent file", func(t *testing.T) {
		_, err := ValidateProject(tmpDir, tmpDir, "does-not-exist.yaml", "local")
		if err == nil {
			t.Error("expected error for non-existent file, got nil")
		}
	})

	// 5. Verify whole workspace validation
	t.Run("workspace scan", func(t *testing.T) {
		results, err := ValidateProject(tmpDir, tmpDir, "", "local")
		if err != nil {
			t.Fatalf("ValidateProject failed: %v", err)
		}

		// We have 4 files: two valid files plus the undefined variable/account cases.
		if len(results) != 4 {
			t.Errorf("expected 4 files validated, got %d", len(results))
		}

		// Find results for each file
		var validRes, stepPrefixedRes, undefVarRes, undefAccRes *ValidationResult
		for i := range results {
			base := filepath.Base(results[i].File)
			switch base {
			case "valid.yaml":
				validRes = &results[i]
			case "step_prefixed.yaml":
				stepPrefixedRes = &results[i]
			case "undefined_var.yaml":
				undefVarRes = &results[i]
			case "undefined_account.yaml":
				undefAccRes = &results[i]
			}
		}

		if validRes == nil || len(validRes.Issues) > 0 {
			t.Errorf("expected valid.yaml to have 0 issues, got: %+v", validRes)
		}
		if stepPrefixedRes == nil || len(stepPrefixedRes.Issues) > 0 {
			t.Errorf("expected step_prefixed.yaml to have 0 issues, got: %+v", stepPrefixedRes)
		}

		if undefVarRes == nil || len(undefVarRes.Issues) != 1 {
			t.Errorf("expected undefined_var.yaml to have exactly 1 issue, got: %+v", undefVarRes)
		} else {
			if undefVarRes.Issues[0].Code != "undefined_variable" {
				t.Errorf("expected undefined_variable code, got: %s", undefVarRes.Issues[0].Code)
			}
		}

		// Since we didn't setup credentials/accounts for "local" env, the unregistered account reference should raise undefined_account
		if undefAccRes == nil || len(undefAccRes.Issues) != 1 {
			t.Errorf("expected undefined_account.yaml to have exactly 1 issue, got: %+v", undefAccRes)
		} else {
			if undefAccRes.Issues[0].Code != "undefined_account" {
				t.Errorf("expected undefined_account code, got: %s", undefAccRes.Issues[0].Code)
			}
		}
	})
}

func TestValidateProjectAcceptsExampleAndComposedOutputVariables(t *testing.T) {
	projectDir := t.TempDir()
	testsDir := filepath.Join(projectDir, ".gherkio", "tests")
	if err := os.MkdirAll(filepath.Join(testsDir, "shared"), 0755); err != nil {
		t.Fatalf("create tests: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, ".gherkio", "config.yaml"), []byte("environments:\n  default: local\n"), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testsDir, "shared", "login.yaml"), []byte(`scenario: Login
steps:
  - request: {method: POST, url: /login}
    save: {authToken: body.token}
`), 0644); err != nil {
		t.Fatalf("write shared test: %v", err)
	}
	if err := os.WriteFile(filepath.Join(testsDir, "main.yaml"), []byte(`scenario: Profiles
examples:
  - role: admin
steps:
  - use: shared/login.yaml
  - request:
      method: GET
      url: /profile/$role
      headers: {Authorization: "Bearer $authToken"}
`), 0644); err != nil {
		t.Fatalf("write main test: %v", err)
	}

	results, err := ValidateProject(projectDir, projectDir, "main.yaml", "")
	if err != nil {
		t.Fatalf("ValidateProject: %v", err)
	}
	if len(results) != 1 || len(results[0].Issues) != 0 {
		t.Fatalf("validation issues = %+v, want none", results)
	}
}
