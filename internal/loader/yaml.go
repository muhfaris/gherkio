package loader

import (
	"gopkg.in/yaml.v3"
)

func yamlUnmarshal(b []byte, v any) error { return yaml.Unmarshal(b, v) }
