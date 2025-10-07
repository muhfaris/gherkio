package loader

import (
	"os"
	"path/filepath"
	"time"
)

type Env struct {
	BaseURL  string            `yaml:"baseURL"`
	Headers  map[string]string `yaml:"headers"`
	Timeouts struct {
		Request string `yaml:"request"`
	} `yaml:"timeouts"`
	Retries struct {
		Max  int    `yaml:"max"`
		Wait string `yaml:"wait"`
	} `yaml:"retries"`
	Redact struct {
		Headers []string `yaml:"headers"`
		JSON    []string `yaml:"json"`
	} `yaml:"redact"`
}

func LoadEnv(dir, name string) (Env, error) {
	p := filepath.Join(dir, name+".yaml")
	b, err := os.ReadFile(p)
	if err != nil {
		return Env{}, err
	}
	var e Env
	if err := yamlUnmarshal(b, &e); err != nil {
		return Env{}, err
	}
	// defaults
	if e.Timeouts.Request == "" {
		e.Timeouts.Request = "20s"
	}
	if e.Retries.Wait == "" {
		e.Retries.Wait = "200ms"
	}
	return e, nil
}

func (e Env) RequestTimeout() time.Duration {
	d, _ := time.ParseDuration(e.Timeouts.Request)
	return d
}

func (e Env) RetryWait() time.Duration {
	d, _ := time.ParseDuration(e.Retries.Wait)
	return d
}
