package model

// TestFile represents a single Gherkio test YAML file.
type TestFile struct {
	Scenario string `yaml:"scenario" json:"scenario" jsonschema:"required,description=The name of the test scenario"`
Tags     []string `yaml:"tags,omitempty" json:"tags,omitempty" jsonschema:"description=Tags for organizing and filtering tests (e.g. smoke auth critical)"`
	Setup    []Step `yaml:"setup,omitempty" json:"setup,omitempty" jsonschema:"description=Pre-condition steps executed before main steps"`
	Steps    []Step `yaml:"steps" json:"steps" jsonschema:"required,description=Main steps to execute for this scenario"`
	Teardown []Step `yaml:"teardown,omitempty" json:"teardown,omitempty" jsonschema:"description=Post-condition steps that always execute, even on failure"`
}

// RetryConfig defines the configuration for a step's retry loop.
type RetryConfig struct {
	Attempts    int    `yaml:"attempts" json:"attempts" jsonschema:"required,description=Number of retry attempts"`
	Interval    int    `yaml:"interval,omitempty" json:"interval,omitempty" jsonschema:"description=Interval in milliseconds between retries"`
	Backoff     string `yaml:"backoff,omitempty" json:"backoff,omitempty" jsonschema:"description=Backoff strategy (e.g. linear, exponential)"`
	MaxDuration string `yaml:"maxDuration,omitempty" json:"maxDuration,omitempty" jsonschema:"description=Maximum duration for the retry loop (e.g. 5s, 1m)"`
	OnStatus    []int  `yaml:"onStatus,omitempty" json:"onStatus,omitempty" jsonschema:"description=List of status codes that trigger a retry"`
}

// TimingConfig holds timing expectations for a step.
type TimingConfig struct {
	Max string `yaml:"max" json:"max" jsonschema:"required,description=Maximum duration the step is allowed to take (e.g. 500ms, 1s)"` // e.g. "500ms", "1s", "250ms"
}

// Step represents a single step in a scenario.
type Step struct {
	Use     string            `yaml:"use,omitempty" json:"use,omitempty" jsonschema:"description=References another scenario file to execute"`
	Request Request           `yaml:"request,omitempty" json:"request,omitempty" jsonschema:"description=HTTP request definition"`
	Retry   *RetryConfig      `yaml:"retry,omitempty" json:"retry,omitempty" jsonschema:"description=Retry configuration for the step"`
	Expect  Expect            `yaml:"expect,omitempty" json:"expect,omitempty" jsonschema:"description=Assertions for the step response"`
	Save    map[string]string `yaml:"save,omitempty" json:"save,omitempty" jsonschema:"description=Variable extractions mapped from response paths"`
	Timing  TimingConfig      `yaml:"timing,omitempty" json:"timing,omitempty" jsonschema:"description=Timing expectations for the step"`
}

// Request represents an HTTP request definition.
type Request struct {
	Service string            `yaml:"service,omitempty" json:"service,omitempty" jsonschema:"description=Name of the service defined in environment"`
	Method  string            `yaml:"method" json:"method" jsonschema:"required,enum=GET,enum=POST,enum=PUT,enum=DELETE,enum=PATCH,description=HTTP method"`
	URL     string            `yaml:"url" json:"url" jsonschema:"required,description=Request URL path or absolute URL. Supports variable interpolation ($var ${var:default} $accounts.(name).(field) $uuid $ulid $randomInt $randomEmail $randomPhone)"`
	Headers map[string]string `yaml:"headers,omitempty" json:"headers,omitempty" jsonschema:"description=HTTP request headers. Supports variable interpolation in values ($var ${var:default} $accounts.(name).(field) $uuid $ulid $randomInt $randomEmail $randomPhone)"`
	Body    interface{}       `yaml:"body,omitempty" json:"body,omitempty" jsonschema:"description=HTTP request body as JSON object or string. Supports variable interpolation in string values ($var ${var:default} $accounts.(name).(field) $uuid $ulid $randomInt $randomEmail $randomPhone)"`
	Timeout string            `yaml:"timeout,omitempty" json:"timeout,omitempty" jsonschema:"description=HTTP client timeout for this request (e.g. 5s 30s 1m)"` // e.g. "5s", "30s", "1m"
}

// Expect holds assertions for a step.
// Status is a special integer assertion; all other keys are string matchers
// (e.g. "body.token": "exists", "jwt.role": "admin").
type Expect struct {
	Status int                    `yaml:"status,omitempty" json:"status,omitempty" jsonschema:"description=Expected HTTP status code"`
	Extra  map[string]interface{} `yaml:",inline" json:"-"`
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
