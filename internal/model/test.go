package model

// TestFile represents a single Gherkio test YAML file.
type TestFile struct {
	Scenario string `yaml:"scenario"`
	Steps    []Step `yaml:"steps"`
}

// TimingConfig holds timing expectations for a step.
type TimingConfig struct {
	Max string `yaml:"max"` // e.g. "500ms", "1s", "250ms"
}

// Step represents a single step in a scenario.
type Step struct {
	Use     string            `yaml:"use,omitempty"`
	Request Request           `yaml:"request,omitempty"`
	Expect  Expect            `yaml:"expect,omitempty"`
	Save    map[string]string `yaml:"save,omitempty"`
	Timing  TimingConfig      `yaml:"timing,omitempty"`
}

// Request represents an HTTP request definition.
type Request struct {
	Service string            `yaml:"service,omitempty"`
	Method  string            `yaml:"method"`
	URL     string            `yaml:"url"`
	Headers map[string]string `yaml:"headers,omitempty"`
	Body    interface{}       `yaml:"body,omitempty"`
}

// Expect holds assertions for a step.
// Status is a special integer assertion; all other keys are string matchers
// (e.g. "body.token": "exists", "jwt.role": "admin").
type Expect struct {
	Status int                    `yaml:"status,omitempty"`
	Extra  map[string]interface{} `yaml:",inline"`
}

// UnmarshalYAML handles the mixed-type expect block.
func (e *Expect) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var raw map[string]interface{}
	if err := unmarshal(&raw); err != nil {
		return err
	}

	e.Extra = make(map[string]interface{})

	for key, val := range raw {
		if key == "status" {
			switch v := val.(type) {
			case int:
				e.Status = v
			default:
				// Try float64 (YAML default numeric type)
				if f, ok := val.(float64); ok {
					e.Status = int(f)
				}
			}
		} else {
			e.Extra[key] = val
		}
	}

	return nil
}
