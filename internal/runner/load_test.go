package runner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunVirtualUsersExecutesSequentialIterationsWithIsolatedState(t *testing.T) {
	projectDir := t.TempDir()
	envDir := filepath.Join(projectDir, ".gherkio", "environments")
	if err := os.MkdirAll(envDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(envDir, "local.yaml"), []byte("baseUrl: https://example.test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	testPath := filepath.Join(projectDir, "load.yaml")
	testYAML := `scenario: Human-readable load run
steps:
  - name: Capture execution identity
    set:
      OBSERVED_VU: $vu
      OBSERVED_ITERATION: $iteration
      CARRY: vu-$vu
`
	if err := os.WriteFile(testPath, []byte(testYAML), 0644); err != nil {
		t.Fatal(err)
	}

	results, err := RunVirtualUsers(RunConfig{
		TestPath:   testPath,
		EnvName:    "local",
		ProjectDir: projectDir,
		StepIndex:  -1,
	}, 2, 3)
	if err != nil {
		t.Fatalf("RunVirtualUsers: %v", err)
	}
	if len(results) != 6 {
		t.Fatalf("got %d results, want 6", len(results))
	}

	for index, result := range results {
		wantVU := index/3 + 1
		wantIteration := index%3 + 1
		if result.VirtualUser != wantVU || result.Iteration != wantIteration || result.IterationsPerUser != 3 {
			t.Errorf("result %d identity = VU %d iteration %d/%d, want VU %d iteration %d/3", index, result.VirtualUser, result.Iteration, result.IterationsPerUser, wantVU, wantIteration)
		}
		if !result.Passed {
			t.Errorf("result %d unexpectedly failed", index)
		}
		if wantIteration > 1 {
			wantCarry := "vu-" + string(rune('0'+wantVU))
			if got := result.ResolvedVars["CARRY"]; got != wantCarry {
				t.Errorf("result %d carried state = %v, want %q", index, got, wantCarry)
			}
		}
	}
}

func TestRunVirtualUsersRejectsExamples(t *testing.T) {
	testPath := filepath.Join(t.TempDir(), "examples.yaml")
	if err := os.WriteFile(testPath, []byte("scenario: conflict\nexamples:\n  - run: 1\nsteps: []\n"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := RunVirtualUsers(RunConfig{TestPath: testPath}, 2, 3)
	if err == nil {
		t.Fatal("expected examples conflict error")
	}
}

func TestCloneVarsDeeplyIsolatesVirtualUsers(t *testing.T) {
	original := map[string]interface{}{
		"object": map[string]interface{}{"id": "one"},
		"array":  []interface{}{map[string]interface{}{"id": "two"}},
	}
	cloned := cloneVars(original)
	cloned["object"].(map[string]interface{})["id"] = "changed"
	cloned["array"].([]interface{})[0].(map[string]interface{})["id"] = "changed"

	if original["object"].(map[string]interface{})["id"] != "one" {
		t.Fatal("nested object was shared between virtual users")
	}
	if original["array"].([]interface{})[0].(map[string]interface{})["id"] != "two" {
		t.Fatal("nested array was shared between virtual users")
	}
}
