package loader

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type Catalog struct {
	Auth      map[string]AuthProfile `yaml:"auth"`
	Endpoints map[string]Endpoint    `yaml:"endpoints"`
}

type AuthProfile struct {
	FromStore   string `yaml:"fromStore"`
	Header      string `yaml:"header"`
	Template    string `yaml:"template"`
	UsernameEnv string `yaml:"usernameEnv"`
	PasswordEnv string `yaml:"passwordEnv"`
}

type Endpoint struct {
	Method  string            `yaml:"method"`
	Path    string            `yaml:"path"`
	Headers map[string]string `yaml:"headers"`
	Auth    string            `yaml:"auth"`
	Expect  *Expectation      `yaml:"expect"`
	Save    map[string]string `yaml:"save"`
}

type Expectation struct {
	Status int `yaml:"status"`
}

func LoadCatalogs(dir string) (Catalog, error) {
	cat := Catalog{Auth: map[string]AuthProfile{}, Endpoints: map[string]Endpoint{}}
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
		var c Catalog
		if err := yamlUnmarshal(b, &c); err != nil {
			return nil
		}
		// merge auth
		for k, v := range c.Auth {
			if _, ok := cat.Auth[k]; ok {
				fmt.Printf("warning: duplicate auth profile %s\n", k)
			}
			cat.Auth[k] = v
		}
		// merge endpoints
		for k, v := range c.Endpoints {
			if _, ok := cat.Endpoints[k]; ok {
				fmt.Printf("warning: duplicate endpoint %s\n", k)
			}
			cat.Endpoints[k] = v
		}
		return nil
	})
	return cat, nil
}
