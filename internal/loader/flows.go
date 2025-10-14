package loader

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type FlowFile struct {
	Flows map[string]Flow `yaml:"flows"`
}

type Flow struct {
	Params []string   `yaml:"params"`
	Steps  []FlowStep `yaml:"steps"`
}

type FlowStep struct {
	Call    string            `yaml:"call,omitempty"`
	Path    map[string]string `yaml:"path,omitempty"`
	Query   map[string]string `yaml:"query,omitempty"`
	Headers map[string]string `yaml:"headers,omitempty"`
	Body    string            `yaml:"body,omitempty"`
	Fixture string            `yaml:"fixture,omitempty"`
	Save    map[string]string `yaml:"save,omitempty"`
	Expect  *Expectation      `yaml:"expect,omitempty"`
	SetAuth string            `yaml:"setAuth,omitempty"`
}

func LoadFlows(dir string) (map[string]Flow, error) {
	flows := map[string]Flow{}
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		name := strings.ToLower(d.Name())
		if !(strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml")) {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		var ff FlowFile
		if err := yamlUnmarshal(b, &ff); err != nil {
			return nil
		}
		for k, v := range ff.Flows {
			flows[k] = v
		}
		return nil
	})
	return flows, nil
}
