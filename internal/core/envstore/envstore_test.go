package envstore

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/muhfaris/gherkio/internal/model"
)

func TestEnvStoreCRUD(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Create a dummy .gherkio setup
	gDir := filepath.Join(tmpDir, ".gherkio")
	envsDir := filepath.Join(gDir, "environments")
	_ = os.MkdirAll(envsDir, 0755)
	_ = os.WriteFile(filepath.Join(gDir, "config.yaml"), []byte(""), 0644)

	// 2. Create environment
	validEnv := &model.Environment{
		BaseURL: "https://api.test.com",
		Services: map[string]model.Service{
			"auth": {BaseURL: "https://auth.test.com"},
		},
	}

	err := Create(tmpDir, "staging", validEnv)
	if err != nil {
		t.Fatalf("Create env failed: %v", err)
	}

	// 3. List environment
	envs, err := List(tmpDir)
	if err != nil {
		t.Fatalf("List envs failed: %v", err)
	}
	if len(envs) != 1 {
		t.Errorf("expected 1 env, got %d", len(envs))
	}
	if envs[0].Name != "staging" || envs[0].BaseURL != "https://api.test.com" {
		t.Errorf("unexpected env details: %+v", envs[0])
	}

	// 4. Update environment
	validEnv.BaseURL = "https://new-api.test.com"
	err = Update(tmpDir, "staging", validEnv)
	if err != nil {
		t.Fatalf("Update env failed: %v", err)
	}

	// 5. Read environment
	readEnv, err := Read(tmpDir, "staging")
	if err != nil {
		t.Fatalf("Read env failed: %v", err)
	}
	if readEnv.BaseURL != "https://new-api.test.com" {
		t.Errorf("expected new-api.test.com, got %s", readEnv.BaseURL)
	}

	// 6. Delete environment
	err = Delete(tmpDir, "staging")
	if err != nil {
		t.Fatalf("Delete env failed: %v", err)
	}
}
