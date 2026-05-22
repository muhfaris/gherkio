package model

// Config represents the .gherkio/config.yaml file.
type Config struct {
	Project      ProjectConfig  `yaml:"project,omitempty"`
	Environments EnvConfig      `yaml:"environments,omitempty"`
	Tests        TestsConfig    `yaml:"tests,omitempty"`
	Security     SecurityConfig `yaml:"security,omitempty"`
	Reports      ReportsConfig  `yaml:"reports,omitempty"`
}

type ProjectConfig struct {
	Name    string `yaml:"name,omitempty"`
	Version string `yaml:"version,omitempty"`
}

type EnvConfig struct {
	Default string `yaml:"default,omitempty"`
	Path    string `yaml:"path,omitempty"`
}

type TestsConfig struct {
	Path string `yaml:"path,omitempty"`
}

type SecurityConfig struct {
	Mask struct {
		Enabled bool     `yaml:"enabled"`
		Fields  []string `yaml:"fields,omitempty"`
	} `yaml:"mask"`
}

type ReportsConfig struct {
	Path          string `yaml:"path,omitempty"`
	Format        string `yaml:"format,omitempty"`
	Archive       bool   `yaml:"archive,omitempty"`
	Retention     int    `yaml:"retention,omitempty"`
	MaskSensitive bool   `yaml:"maskSensitive,omitempty"`
}
