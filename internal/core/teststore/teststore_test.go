package teststore

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/muhfaris/gherkio/internal/model"
)

func TestTestStoreCRUDAndValidate(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Create a dummy .gherkio setup
	gDir := filepath.Join(tmpDir, ".gherkio")
	testsDir := filepath.Join(gDir, "tests")
	schemasDir := filepath.Join(gDir, "schemas")
	_ = os.MkdirAll(testsDir, 0755)
	_ = os.MkdirAll(schemasDir, 0755)
	_ = os.WriteFile(filepath.Join(gDir, "config.yaml"), []byte(""), 0644)

	// 2. Validate scenarios
	validTest := &model.TestFile{
		Scenario: "Valid Test",
		Steps: []model.Step{
			{
				Request: model.Request{
					Method: "POST",
					URL:    "/auth/login",
				},
			},
		},
	}

	res, err := Validate(validTest, schemasDir)
	if err != nil || !res.Valid {
		t.Fatalf("validation failed unexpectedly: %v (res: %+v)", err, res)
	}

	invalidTest := &model.TestFile{
		Scenario: "", // invalid scenario
		Steps: []model.Step{
			{
				Request: model.Request{
					Method: "INVALID", // invalid method
					URL:    "",        // missing URL
				},
			},
		},
	}

	res, err = Validate(invalidTest, schemasDir)
	if err != nil {
		t.Fatalf("Validate error: %v", err)
	}
	if res.Valid {
		t.Error("expected validation to fail for invalid scenario structure")
	}

	// 3. Create Test
	err = CreateTest(testsDir, schemasDir, "auth/login.yaml", validTest)
	if err != nil {
		t.Fatalf("CreateTest failed: %v", err)
	}

	// Double check list
	list, err := ListTests(testsDir)
	if err != nil {
		t.Fatalf("ListTests failed: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected 1 test, got %d", len(list))
	}
	if list[0].RelativePath != "auth/login.yaml" {
		t.Errorf("expected relPath auth/login.yaml, got %s", list[0].RelativePath)
	}

	// 4. Update Test
	validTest.Scenario = "Updated Valid Test"
	err = UpdateTest(filepath.Join(testsDir, "auth/login.yaml"), validTest, schemasDir)
	if err != nil {
		t.Fatalf("UpdateTest failed: %v", err)
	}

	// 5. Delete Test
	err = DeleteTest(filepath.Join(testsDir, "auth/login.yaml"))
	if err != nil {
		t.Fatalf("DeleteTest failed: %v", err)
	}
}
