package project

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	dirPerm os.FileMode = 0755
)

// defaultLocalEnvTemplate is the default environment file content.
const defaultLocalEnvTemplate = `baseUrl: https://dummyjson.com
services:
  auth:
    baseUrl: http://localhost:3001
`

// defaultExampleTestTemplate is the default example test file content.
const defaultExampleTestTemplate = `scenario: login example

steps:
  - request:
      method: POST
      url: /auth/login

      body:
        username: $username
        password: $password
        expiresInMins: 30

    expect:
      status: 200
      body.accessToken: exists
      body.refreshToken: exists
      body.email: email
      schema: example/login-response

    save:
      accessToken: body.accessToken
      refreshToken: body.refreshToken
`

// defaultExampleMeTemplate is the default example test for auth/me.
const defaultExampleMeTemplate = `scenario: get current user profile

steps:
  - use: login.yaml
  - request:
      method: GET
      url: /auth/me
      headers:
        Authorization: Bearer $accessToken
    expect:
      status: 200
      body.id: exists
      body.username: $username
      schema: example/user-response
`

// defaultExampleRefreshTemplate is the default example test for auth/refresh.
const defaultExampleRefreshTemplate = `scenario: refresh token

steps:
  - use: me.yaml
  - request:
      method: POST
      url: /auth/refresh
      headers:
        Content-Type: application/json
      body:
        refreshToken: $refreshToken
        expiresInMins: 30
    expect:
      status: 200
      body.accessToken: exists
      body.refreshToken: exists
`

// defaultExampleBuiltinsTemplate shows how to use built-in generator variables ($uuid, $ulid, $randomInt).
const defaultExampleBuiltinsTemplate = `scenario: using built-in generator variables

steps:
  - request:
      method: POST
      url: /auth/login

      body:
        username: $accounts.alice.username
        password: $accounts.alice.password
        expiresInMins: 30
        idempotencyKey: $uuid
        requestId: $ulid
        otpCode: $randomInt
        email: $randomEmail
        phone: $randomPhone

    expect:
      status: 200
      body.accessToken: exists
      body.refreshToken: exists
      schema: example/login-response

    save:
      accessToken: body.accessToken

  - request:
      method: GET
      url: /auth/me
      headers:
        Authorization: Bearer $accessToken
        X-Idempotency: $uuid
    expect:
      status: 200
      body.username: $accounts.alice.username
      schema: example/user-response
`

// defaultExampleAccountsTemplate shows how to use $accounts.<name>.<field> to access
// any account's credentials directly without needing the --account flag.
const defaultExampleAccountsTemplate = `scenario: login as alice via $accounts variable

steps:
  - request:
      method: POST
      url: /auth/login

      body:
        username: $accounts.alice.username
        password: $accounts.alice.password
        expiresInMins: 30

    expect:
      status: 200
      body.accessToken: exists
      body.refreshToken: exists
      body.email: email
      schema: example/login-response

    save:
      accessToken: body.accessToken
      refreshToken: body.refreshToken

  - request:
      method: GET
      url: /auth/me
      headers:
        Authorization: Bearer $accessToken
    expect:
      status: 200
      body.username: $accounts.alice.username
      schema: example/user-response
`

// defaultExampleMultipartTemplate shows how to do multipart form-data uploads.
const defaultExampleMultipartTemplate = `scenario: upload user avatar example

steps:
  - request:
      method: POST
      url: /users/1/avatar
      multipart:
        fields:
          description: "Profile avatar"
          userId: "1"
        files:
          avatar:
            path: .gherkio/config.yaml
            contentType: text/yaml
            filename: avatar-config.yaml
    expect:
      status: 200
`

// defaultConfigTemplate is the default config.yaml template content.
func defaultConfigTemplate(version string) string {
	if version == "dev" || version == "" {
		version = "0.1.0"
	}
	return fmt.Sprintf(`# ======================================================================
# Gherkio Declarative API Testing Platform — Configuration Map
# ======================================================================
# Use this file to define project metadata, test paths, HTML reporting,
# custom credential masking, and request sandboxing.

# Gherkio tool version that initialized this project config.
gherkio_version: %s

# ----------------------------------------------------------------------
# 1. Project Metadata
# ----------------------------------------------------------------------
project:
  name: "my-api-testing-suite"
  version: "1.0.0"

# ----------------------------------------------------------------------
# 2. Environments Directory Configuration
# ----------------------------------------------------------------------
environments:
  # The default environment name to target if --env is omitted.
  default: local
  # Path to target environments folder containing baseUrl mappings.
  path: .gherkio/environments

# ----------------------------------------------------------------------
# 3. Tests & Scenarios Configuration
# ----------------------------------------------------------------------
tests:
  # Directory where declarative YAML scenario test files reside.
  path: .gherkio/tests

# ----------------------------------------------------------------------
# 4. JSON Schemas Configuration
# ----------------------------------------------------------------------
schemas:
  # Directory containing validation files for schema matchers.
  path: .gherkio/schemas

# ----------------------------------------------------------------------
# 5. Multipart Assets Configuration
# ----------------------------------------------------------------------
assets:
  # Default directory used for multipart files declared with a relative name.
  path: assets

# ----------------------------------------------------------------------
# 6. Reports & Core Failure Snapshots
# ----------------------------------------------------------------------
reports:
  # Output path where execution summaries are compiled.
  path: .gherkio/reports
  # Report output format (options: "html", "json", "html,json" for both).
  format: html
  # Archive previous report results in .gherkio/reports/archive/.
  archive: true
  # Number of past report archives to retain before cleaning up disk.
  retention: 10
  # Mask sensitive fields (e.g. Bearer tokens) inside public HTML reports.
  maskSensitive: true

  # failure-debug-snapshots:
  failures:
    # Set to true to automatically write JSON debugging snapshots on any step fail.
    enabled: true
    # Folder path where individual core JSON dumps will reside.
    path: .gherkio/reports/failures
    # Apply global credentials masking to variables/payloads inside failure dumps.
    maskSensitive: true
    # Maximum count of individual failure dumps to retain (prevents disk bloat).
    retainCount: 50

# ----------------------------------------------------------------------
# 7. Security, Masking, and Domain Sandboxing Guardrails
# ----------------------------------------------------------------------
security:
  # Sensitive field masking to secure stdout logs and run traces.
  mask:
    # Enable automatic sensitive credential variable masking.
    enabled: true
    # Custom sensitive dictionary words to mask in execution outputs (case-insensitive).
    fields:
      - secret
      - password
      - token
      - auth
      - authorization
      - signature
      - apikey
      - api-key

  # sandboxing:
  sandboxing:
    # Set to true to restrict request scopes to allowed domains.
    enabled: false
    # Allowed domains pattern matching (supports wildcards, empty allows all).
    allowedDomains:
      - "*.api.dummyjson.com"
      - "localhost:*"
      - "127.0.0.1:*"
    # Domains that are explicitly blocked even if they would match allowed lists.
    blockedDomains:
      - "*.malicious.com"
      - "untrusted.org"
    # Block local loopback, link-local, and RFC1918 private subnets.
    blockPrivateSubnets: true
`, version)
}

// defaultExampleLoginSchemaTemplate is the default example schema for login response.
const defaultExampleLoginSchemaTemplate = `type: object
required:
  - accessToken
  - refreshToken
  - id
  - username
  - email
properties:
  accessToken:
    type: string
  refreshToken:
    type: string
  id:
    type: integer
  username:
    type: string
  email:
    type: string
    format: email
  firstName:
    type: string
  lastName:
    type: string
  gender:
    type: string
  image:
    type: string
`

// defaultExampleUserSchemaTemplate is the default example schema for user response.
const defaultExampleUserSchemaTemplate = `type: object
required:
  - id
  - username
  - email
properties:
  id:
    type: integer
  username:
    type: string
  email:
    type: string
    format: email
  firstName:
    type: string
  lastName:
    type: string
  gender:
    type: string
  image:
    type: string
`

// Initialize sets up a new Gherkio project structure in the target projectDir.
func Initialize(projectDir string, version string) error {
	baseDir := filepath.Join(projectDir, ".gherkio")

	// Directories to create
	dirs := []string{
		baseDir,
		filepath.Join(baseDir, "environments"),
		filepath.Join(baseDir, "credentials"),
		filepath.Join(baseDir, "tests"),
		filepath.Join(baseDir, "reports"),
		filepath.Join(baseDir, "reports", "latest"),
		filepath.Join(baseDir, "reports", "archive"),
		filepath.Join(baseDir, "reports", "failures"), // 🚨 NEW: Failures folder for RFC-18 snapshots
		filepath.Join(baseDir, "schemas"),
	}

	fmt.Fprintf(os.Stderr, "Creating .gherkio project structure in %s...\n", projectDir)

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, dirPerm); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
		fmt.Fprintf(os.Stderr, "  📁  %s\n", dir)
	}

	// Create default config.yaml
	configPath := filepath.Join(baseDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(defaultConfigTemplate(version)), 0644); err != nil {
		return fmt.Errorf("failed to write config.yaml: %w", err)
	}
	fmt.Fprintf(os.Stderr, "  📄  %s\n", configPath)

	// Create default environment file
	envPath := filepath.Join(baseDir, "environments", "local.yaml")
	if err := os.WriteFile(envPath, []byte(defaultLocalEnvTemplate), 0644); err != nil {
		return fmt.Errorf("failed to write environment file: %w", err)
	}
	fmt.Fprintf(os.Stderr, "  📄  %s\n", envPath)

	// Create example test file
	exampleDir := filepath.Join(baseDir, "tests", "example", "auth")
	if err := os.MkdirAll(exampleDir, dirPerm); err != nil {
		return fmt.Errorf("failed to create example tests directory: %w", err)
	}
	fmt.Fprintf(os.Stderr, "  📁  %s\n", exampleDir)

	testPath := filepath.Join(exampleDir, "login.yaml")
	if err := os.WriteFile(testPath, []byte(defaultExampleTestTemplate), 0644); err != nil {
		return fmt.Errorf("failed to write example test file: %w", err)
	}
	fmt.Fprintf(os.Stderr, "  📄  %s\n", testPath)

	mePath := filepath.Join(exampleDir, "me.yaml")
	if err := os.WriteFile(mePath, []byte(defaultExampleMeTemplate), 0644); err != nil {
		return fmt.Errorf("failed to write me test file: %w", err)
	}
	fmt.Fprintf(os.Stderr, "  📄  %s\n", mePath)

	refreshPath := filepath.Join(exampleDir, "refresh.yaml")
	if err := os.WriteFile(refreshPath, []byte(defaultExampleRefreshTemplate), 0644); err != nil {
		return fmt.Errorf("failed to write refresh test file: %w", err)
	}
	fmt.Fprintf(os.Stderr, "  📄  %s\n", refreshPath)

	// Create built-in variables example showing $uuid, $ulid, $randomInt
	builtinsExampleDir := filepath.Join(baseDir, "tests", "example", "builtins")
	if err := os.MkdirAll(builtinsExampleDir, dirPerm); err != nil {
		return fmt.Errorf("failed to create builtins example directory: %w", err)
	}
	fmt.Fprintf(os.Stderr, "  📁  %s\n", builtinsExampleDir)

	builtinsPath := filepath.Join(builtinsExampleDir, "login-with-generators.yaml")
	if err := os.WriteFile(builtinsPath, []byte(defaultExampleBuiltinsTemplate), 0644); err != nil {
		return fmt.Errorf("failed to write builtins example test file: %w", err)
	}
	fmt.Fprintf(os.Stderr, "  📄  %s\n", builtinsPath)

	// Create accounts example showing $accounts.<name>.<field> access
	accountsExampleDir := filepath.Join(baseDir, "tests", "example", "accounts")
	if err := os.MkdirAll(accountsExampleDir, dirPerm); err != nil {
		return fmt.Errorf("failed to create accounts example directory: %w", err)
	}
	fmt.Fprintf(os.Stderr, "  📁  %s\n", accountsExampleDir)

	accountsPath := filepath.Join(accountsExampleDir, "login-as-alice.yaml")
	if err := os.WriteFile(accountsPath, []byte(defaultExampleAccountsTemplate), 0644); err != nil {
		return fmt.Errorf("failed to write accounts example test file: %w", err)
	}
	fmt.Fprintf(os.Stderr, "  📄  %s\n", accountsPath)

	// Create multipart upload example
	multipartExampleDir := filepath.Join(baseDir, "tests", "example", "multipart")
	if err := os.MkdirAll(multipartExampleDir, dirPerm); err != nil {
		return fmt.Errorf("failed to create multipart example directory: %w", err)
	}
	fmt.Fprintf(os.Stderr, "  📁  %s\n", multipartExampleDir)

	multipartPath := filepath.Join(multipartExampleDir, "upload.yaml")
	if err := os.WriteFile(multipartPath, []byte(defaultExampleMultipartTemplate), 0644); err != nil {
		return fmt.Errorf("failed to write multipart example test file: %w", err)
	}
	fmt.Fprintf(os.Stderr, "  📄  %s\n", multipartPath)

	// Create example schema files
	schemaExampleDir := filepath.Join(baseDir, "schemas", "example")
	if err := os.MkdirAll(schemaExampleDir, dirPerm); err != nil {
		return fmt.Errorf("failed to create example schema directory: %w", err)
	}
	fmt.Fprintf(os.Stderr, "  📁  %s\n", schemaExampleDir)

	loginSchemaPath := filepath.Join(schemaExampleDir, "login-response.yaml")
	if err := os.WriteFile(loginSchemaPath, []byte(defaultExampleLoginSchemaTemplate), 0644); err != nil {
		return fmt.Errorf("failed to write login-response schema file: %w", err)
	}
	fmt.Fprintf(os.Stderr, "  📄  %s\n", loginSchemaPath)

	userSchemaPath := filepath.Join(schemaExampleDir, "user-response.yaml")
	if err := os.WriteFile(userSchemaPath, []byte(defaultExampleUserSchemaTemplate), 0644); err != nil {
		return fmt.Errorf("failed to write user-response schema file: %w", err)
	}
	fmt.Fprintf(os.Stderr, "  📄  %s\n", userSchemaPath)

	// Create default credentials file for local environment
	defaultLocalCredsTemplate := `# Local environment credentials
# These credentials are used when running tests with --env local
# Replace with your actual test credentials

accounts:
  default:
    username: emilys
    password: emilyspass
    role: user

  alice:
    username: "alice"
    password: "alicepass"
    role: user

  # admin:
  #   username: admin@example.com
  #   password: admin-secret
  #   role: admin
`
	credsPath := filepath.Join(baseDir, "credentials", "local.yaml")
	if err := os.WriteFile(credsPath, []byte(defaultLocalCredsTemplate), 0644); err != nil {
		return fmt.Errorf("failed to write credentials file: %w", err)
	}
	fmt.Fprintf(os.Stderr, "  📄  %s\n", credsPath)

	fmt.Fprintln(os.Stderr, "\n✨ Gherkio project initialized successfully!")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "  Quick start:")
	fmt.Fprintln(os.Stderr, "    gherkio run example/auth/me.yaml")
	fmt.Fprintln(os.Stderr, "    gherkio run example/auth/refresh.yaml")
	fmt.Fprintln(os.Stderr, "    gherkio run example/auth/me.yaml --verbose")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "  Multi-account (no --account flag needed):")
	fmt.Fprintln(os.Stderr, "    gherkio run example/accounts/login-as-alice.yaml")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "  Built-in generators ($uuid, $ulid, $randomInt):")
	fmt.Fprintln(os.Stderr, "    gherkio run example/builtins/login-with-generators.yaml")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "  Multipart upload:")
	fmt.Fprintln(os.Stderr, "    gherkio run example/multipart/upload.yaml")

	return nil
}
