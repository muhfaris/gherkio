package model

// Config represents the .gherkio/config.yaml file.
type Config struct {
	GherkioVersion string         `yaml:"gherkio_version,omitempty" jsonschema:"description=Gherkio tool version that created this project"`
	Project        ProjectConfig  `yaml:"project,omitempty" jsonschema:"description=Project metadata"`
	Environments   EnvConfig      `yaml:"environments,omitempty" jsonschema:"description=Environment configuration"`
	Tests          TestsConfig    `yaml:"tests,omitempty" jsonschema:"description=Test path configuration"`
	Schemas        SchemasConfig  `yaml:"schemas,omitempty" jsonschema:"description=Schema directory path"`
	Security       SecurityConfig `yaml:"security,omitempty" jsonschema:"description=Security and masking configuration"`
	Reports        ReportsConfig  `yaml:"reports,omitempty" jsonschema:"description=Report generation configuration"`
}

type ProjectConfig struct {
	Name    string `yaml:"name,omitempty" jsonschema:"description=Project name"`
	Version string `yaml:"version,omitempty" jsonschema:"description=Project version"`
}

type EnvConfig struct {
	Default string `yaml:"default,omitempty" jsonschema:"description=Default environment name"`
	Path    string `yaml:"path,omitempty" jsonschema:"description=Path to environments directory"`
}

type TestsConfig struct {
	Path string `yaml:"path,omitempty" jsonschema:"description=Path to tests directory"`
}

type SchemasConfig struct {
	Path string `yaml:"path,omitempty" jsonschema:"description=Path to schemas directory"`
}

type SecurityConfig struct {
	Mask struct {
		Enabled bool     `yaml:"enabled" jsonschema:"description=Enable sensitive field masking"`
		Fields  []string `yaml:"fields,omitempty" jsonschema:"description=Custom field names to mask (case-insensitive)"`
	} `yaml:"mask" jsonschema:"description=Mask configuration"`
	Sandboxing SandboxConfig `yaml:"sandboxing,omitempty" jsonschema:"description=Sandboxing and outbound network configuration"`
	Sandbox    SandboxConfig `yaml:"sandbox,omitempty" jsonschema:"description=Sandboxing and outbound network configuration"`
}

type SandboxConfig struct {
	Enabled             bool     `yaml:"enabled" jsonschema:"description=Enable domain/IP sandboxing"`
	AllowedDomains      []string `yaml:"allowedDomains,omitempty" jsonschema:"description=List of allowed domains"`
	BlockedDomains      []string `yaml:"blockedDomains,omitempty" jsonschema:"description=List of explicitly blocked domains"`
	BlockPrivateSubnets bool     `yaml:"blockPrivateSubnets,omitempty" jsonschema:"description=Block local/private IP ranges"`
}

type ReportsConfig struct {
	Path          string `yaml:"path,omitempty" jsonschema:"description=Output path for reports"`
	Format        string `yaml:"format,omitempty" jsonschema:"enum=html,enum=json,description=Report format (html or json)"`
	Archive       bool   `yaml:"archive,omitempty" jsonschema:"description=Archive previous reports"`
	Retention     int    `yaml:"retention,omitempty" jsonschema:"description=Number of archives to retain,default=10"`
	MaskSensitive bool   `yaml:"maskSensitive,omitempty" jsonschema:"description=Mask sensitive data in reports"`
	Failures      FailureConfig `yaml:"failures,omitempty" jsonschema:"description=Failure debug snapshot configuration"`
}

// FailureConfig defines the configuration for automated failure snapshot generation.
type FailureConfig struct {
	Enabled       bool   `yaml:"enabled,omitempty" jsonschema:"description=Enable automated failure snapshot generation"`
	Path          string `yaml:"path,omitempty" jsonschema:"description=Output path for failure snapshots"`
	MaskSensitive bool   `yaml:"maskSensitive,omitempty" jsonschema:"description=Mask sensitive data in failure snapshots"`
	RetainCount   int    `yaml:"retainCount,omitempty" jsonschema:"description=Maximum number of failure snapshots to retain,default=50"`
}
