package runner

import (
	"fmt"
	"sync"
)

// RunVirtualUsers executes one test file with isolated virtual-user state.
// Each virtual user runs its assigned iterations sequentially; virtual users
// execute concurrently. Results are returned in VU/iteration order.
func RunVirtualUsers(cfg RunConfig, virtualUsers, iterationsPerUser int) ([]*RunResult, error) {
	if virtualUsers < 1 {
		return nil, fmt.Errorf("virtual users must be at least 1")
	}
	if iterationsPerUser < 1 {
		return nil, fmt.Errorf("iterations per user must be at least 1")
	}

	testFile, err := LoadTestFile(cfg.TestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load test file: %w", err)
	}
	if len(testFile.Examples) > 0 {
		return nil, fmt.Errorf("virtual-user mode cannot be combined with examples; use one execution model per test")
	}

	seedSession := cloneVars(cfg.SessionVars)
	if cfg.SessionFile != "" {
		loaded, loadErr := LoadSessionVars(cfg.SessionFile)
		if loadErr == nil {
			for key, value := range loaded {
				seedSession[key] = value
			}
		}
	}

	results := make([]*RunResult, virtualUsers*iterationsPerUser)
	var wg sync.WaitGroup
	var errMu sync.Mutex
	var firstErr error

	for vu := 1; vu <= virtualUsers; vu++ {
		wg.Add(1)
		go func(vu int) {
			defer wg.Done()
			vuSession := cloneVars(seedSession)
			vuCredentials := cloneVars(cfg.CredentialVars)
			vuAccounts := cloneVars(cfg.AllAccounts)
			for iteration := 1; iteration <= iterationsPerUser; iteration++ {
				iterationCfg := cfg
				iterationCfg.SessionFile = ""
				iterationCfg.SessionVars = vuSession
				iterationCfg.CredentialVars = vuCredentials
				iterationCfg.AllAccounts = vuAccounts
				iterationCfg.VirtualUser = vu
				iterationCfg.WorkflowIteration = iteration
				iterationCfg.IterationsPerUser = iterationsPerUser

				result, runErr := Run(iterationCfg)
				if runErr != nil {
					errMu.Lock()
					if firstErr == nil {
						firstErr = fmt.Errorf("virtual user %d iteration %d: %w", vu, iteration, runErr)
					}
					errMu.Unlock()
					return
				}
				results[(vu-1)*iterationsPerUser+(iteration-1)] = result
			}
		}(vu)
	}
	wg.Wait()

	if firstErr != nil {
		return results, firstErr
	}
	return results, nil
}

func cloneVars(source map[string]interface{}) map[string]interface{} {
	cloned := make(map[string]interface{}, len(source))
	for key, value := range source {
		cloned[key] = cloneVarValue(value)
	}
	return cloned
}

func cloneVarValue(value interface{}) interface{} {
	switch typed := value.(type) {
	case map[string]interface{}:
		return cloneVars(typed)
	case map[string]string:
		cloned := make(map[string]string, len(typed))
		for key, item := range typed {
			cloned[key] = item
		}
		return cloned
	case []interface{}:
		cloned := make([]interface{}, len(typed))
		for index, item := range typed {
			cloned[index] = cloneVarValue(item)
		}
		return cloned
	case []string:
		return append([]string(nil), typed...)
	default:
		return value
	}
}
