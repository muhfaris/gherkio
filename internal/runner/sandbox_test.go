package runner

import (
	"strings"
	"testing"

	"github.com/muhfaris/gherkio/internal/model"
)

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name       string
		rawURL     string
		cfg        *model.SandboxConfig
		wantError  bool
		errContain string
	}{
		{
			name:      "nil config allows everything",
			rawURL:    "http://127.0.0.1/foo",
			cfg:       nil,
			wantError: false,
		},
		{
			name:   "disabled config allows everything",
			rawURL: "http://127.0.0.1/foo",
			cfg: &model.SandboxConfig{
				Enabled: false,
			},
			wantError: false,
		},
		{
			name:   "blockPrivateSubnets loopback IPv4",
			rawURL: "http://127.0.0.1/foo",
			cfg: &model.SandboxConfig{
				Enabled:             true,
				BlockPrivateSubnets: true,
			},
			wantError:  true,
			errContain: "private or local loopback IP",
		},
		{
			name:   "blockPrivateSubnets localhost string fallback",
			rawURL: "http://localhost:8080/foo",
			cfg: &model.SandboxConfig{
				Enabled:             true,
				BlockPrivateSubnets: true,
			},
			wantError:  true,
			errContain: "private or local loopback IP",
		},
		{
			name:   "blockPrivateSubnets private subnet class C",
			rawURL: "http://192.168.1.5:9000/bar",
			cfg: &model.SandboxConfig{
				Enabled:             true,
				BlockPrivateSubnets: true,
			},
			wantError:  true,
			errContain: "private or local loopback IP",
		},
		{
			name:   "blockPrivateSubnets Link-Local AWS metadata",
			rawURL: "http://169.254.169.254/latest/meta-data/",
			cfg: &model.SandboxConfig{
				Enabled:             true,
				BlockPrivateSubnets: true,
			},
			wantError:  true,
			errContain: "private or local loopback IP",
		},
		{
			name:   "allowed domains success exact match",
			rawURL: "https://api.dummyjson.com/products",
			cfg: &model.SandboxConfig{
				Enabled:        true,
				AllowedDomains: []string{"api.dummyjson.com"},
			},
			wantError: false,
		},
		{
			name:   "allowed domains success wildcard",
			rawURL: "https://foo.api.dummyjson.com/products",
			cfg: &model.SandboxConfig{
				Enabled:        true,
				AllowedDomains: []string{"*.api.dummyjson.com"},
			},
			wantError: false,
		},
		{
			name:   "allowed domains failure not matched",
			rawURL: "https://malicious.com/products",
			cfg: &model.SandboxConfig{
				Enabled:        true,
				AllowedDomains: []string{"*.api.dummyjson.com"},
			},
			wantError:  true,
			errContain: "not in the allowed domains list",
		},
		{
			name:   "blocked domains exact match",
			rawURL: "https://untrusted.org/hack",
			cfg: &model.SandboxConfig{
				Enabled:        true,
				BlockedDomains: []string{"untrusted.org"},
			},
			wantError:  true,
			errContain: "is explicitly blocked",
		},
		{
			name:   "blocked domains wildcard",
			rawURL: "https://bad.malicious.com/hack",
			cfg: &model.SandboxConfig{
				Enabled:        true,
				BlockedDomains: []string{"*.malicious.com"},
			},
			wantError:  true,
			errContain: "is explicitly blocked",
		},
		{
			name:   "blocked domain takes precedence over allowed wildcard",
			rawURL: "https://malicious.api.dummyjson.com/hack",
			cfg: &model.SandboxConfig{
				Enabled:        true,
				AllowedDomains: []string{"*.api.dummyjson.com"},
				BlockedDomains: []string{"malicious.api.dummyjson.com"},
			},
			wantError:  true,
			errContain: "is explicitly blocked",
		},
		{
			name:   "port strip matching localhost",
			rawURL: "http://localhost:5000/auth",
			cfg: &model.SandboxConfig{
				Enabled:        true,
				AllowedDomains: []string{"localhost:*"},
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.rawURL, tt.cfg)
			if tt.wantError {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errContain)
				}
				if !strings.Contains(err.Error(), tt.errContain) {
					t.Errorf("expected error containing %q, got: %v", tt.errContain, err)
				}
			} else {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
			}
		})
	}
}
