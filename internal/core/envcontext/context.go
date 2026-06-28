package envcontext

import (
	"fmt"

	"github.com/muhfaris/gherkio/internal/core/credentialstore"
	"github.com/muhfaris/gherkio/internal/core/envstore"
	"github.com/muhfaris/gherkio/internal/core/project"
)

// EnvContext represents the unified environment context for a Gherkio project.
type EnvContext struct {
	ProjectRoot string                      `json:"projectRoot"`
	Environments []EnvironmentInfo          `json:"environments"`
	Accounts    map[string][]string        `json:"accounts"`
	AutoSelect  *AutoSelectHint            `json:"autoSelect,omitempty"`
}

// EnvironmentInfo details about a single environment.
type EnvironmentInfo struct {
	Name          string `json:"name"`
	BaseURL       string `json:"baseUrl"`
	ServicesCount int    `json:"servicesCount"`
	Path          string `json:"path"`
}

// AutoSelectHint provides hints for automatic selection when there's only one option.
type AutoSelectHint struct {
	EnvName     string `json:"env,omitempty"`
	AccountName string `json:"account,omitempty"`
	Reason      string `json:"reason"`
}

// GetContext returns the unified environment context for a project.
func GetContext(projectDir string) (*EnvContext, error) {
	// Get project metadata
	meta, err := project.GetMeta(projectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get project metadata: %w", err)
	}

	ctx := &EnvContext{
		ProjectRoot: projectDir,
		Environments: []EnvironmentInfo{},
		Accounts:    make(map[string][]string),
	}

	// List all environments
	envs, err := envstore.List(projectDir)
	if err != nil {
		return nil, fmt.Errorf("failed to list environments: %w", err)
	}

	for _, env := range envs {
		envInfo := EnvironmentInfo{
			Name:          env.Name,
			BaseURL:       env.BaseURL,
			ServicesCount: env.ServicesCount,
			Path:          meta.EnvsDir + "/" + env.Name + ".yaml",
		}
		ctx.Environments = append(ctx.Environments, envInfo)
	}

	// List accounts for each environment
	creds, err := credentialstore.List(projectDir)
	if err != nil {
		// Non-fatal: credentials may not exist
		creds = nil
	}

	credMap := make(map[string][]string)
	if creds != nil {
		for _, c := range creds {
			credMap[c.EnvName] = c.Accounts
		}
	}

	// Populate accounts for all environments (even if no credentials)
	for _, env := range ctx.Environments {
		if _, exists := credMap[env.Name]; !exists {
			ctx.Accounts[env.Name] = []string{}
		} else {
			ctx.Accounts[env.Name] = credMap[env.Name]
		}
	}

	// Compute auto-selection hints
	ctx.AutoSelect = computeAutoSelect(ctx.Environments, ctx.Accounts)

	return ctx, nil
}

// computeAutoSelect determines if there's a single env/account that should be auto-selected.
func computeAutoSelect(envs []EnvironmentInfo, accounts map[string][]string) *AutoSelectHint {
	hint := &AutoSelectHint{}

	switch {
	case len(envs) == 0:
		hint.Reason = "no environments configured"
		return hint

	case len(envs) == 1:
		envName := envs[0].Name
		hint.EnvName = envName
		accs, hasAccounts := accounts[envName]

		if !hasAccounts || len(accs) == 0 {
			hint.Reason = "single environment with no accounts"
			return hint
		}

		if len(accs) == 1 {
			hint.AccountName = accs[0]
			hint.Reason = "single environment and single account"
			return hint
		}

		hint.Reason = "single environment (multiple accounts available)"
		return hint

	default:
		// Multiple environments - check if any env has only one account
		singleAccountEnvs := []string{}
		for _, env := range envs {
			accs, hasAccounts := accounts[env.Name]
			if hasAccounts && len(accs) == 1 {
				singleAccountEnvs = append(singleAccountEnvs, env.Name)
			}
		}

		if len(singleAccountEnvs) == 1 {
			hint.EnvName = singleAccountEnvs[0]
			hint.AccountName = accounts[singleAccountEnvs[0]][0]
			hint.Reason = "only one environment has a single account"
			return hint
		}

		hint.Reason = "multiple environments available"
		return hint
	}
}
