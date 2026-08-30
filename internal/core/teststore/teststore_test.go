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

func TestValidate_TransformConfig(t *testing.T) {
	tmpDir := t.TempDir()
	schemasDir := filepath.Join(tmpDir, ".gherkio", "schemas")

	// Invalid projection config: from doesn't start with $, limit is negative, select is empty
	invalidTest := &model.TestFile{
		Scenario: "Invalid Transform Test",
		Steps: []model.Step{
			{
				Request: model.Request{
					Method: "POST",
					URL:    "/auth/login",
					Transform: map[string]*model.ProjectionConfig{
						"survey.questions": {
							From:   "raw_questions", // invalid: must start with $
							Limit:  -1,              // invalid: must be positive
							Select: nil,             // invalid: select is required
						},
					},
				},
			},
		},
	}

	res, err := Validate(invalidTest, schemasDir)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	if res.Valid {
		t.Fatal("Expected validation to fail, but it passed")
	}

	expectedErrors := map[string]string{
		"steps[0].request.transform.survey.questions.from":   "invalid_variable_reference",
		"steps[0].request.transform.survey.questions.limit":  "invalid_limit",
		"steps[0].request.transform.survey.questions.select": "missing_field",
	}

	for _, e := range res.Errors {
		if expectedCode, exists := expectedErrors[e.Field]; exists {
			if e.Code != expectedCode {
				t.Errorf("For field %q, expected code %q, got %q", e.Field, expectedCode, e.Code)
			}
			delete(expectedErrors, e.Field)
		}
	}

	if len(expectedErrors) > 0 {
		t.Errorf("Some expected errors were not found: %+v", expectedErrors)
	}
}

func TestValidate_RepeatAndNestedSteps(t *testing.T) {
	test := &model.TestFile{
		Scenario: "Repeated lookup",
		Steps: []model.Step{{
			Repeat: &model.RepeatConfig{
				Attempts: 3,
				Until:    "$count == 0",
				Steps: []model.Step{{
					Request: model.Request{Method: "GET", URL: "/items"},
				}},
			},
		}},
	}

	result, err := Validate(test, "")
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if !result.Valid {
		t.Fatalf("repeat validation failed: %+v", result.Errors)
	}

	test.Steps[0].Repeat.Steps[0].Request.Method = "INVALID"
	result, err = Validate(test, "")
	if err != nil {
		t.Fatalf("Validate nested invalid request: %v", err)
	}
	if result.Valid {
		t.Fatal("invalid nested request unexpectedly passed validation")
	}
}

func TestValidateAcceptsNegatedSchemaReference(t *testing.T) {
	schemasDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(schemasDir, "error.yaml"), []byte("type: object\n"), 0644); err != nil {
		t.Fatalf("write schema: %v", err)
	}
	test := &model.TestFile{
		Scenario: "Reject error schema",
		Steps: []model.Step{{
			Request: model.Request{Method: "GET", URL: "/items"},
			Expect:  model.Expect{Extra: map[string]interface{}{"schema": "not error"}},
		}},
	}
	result, err := Validate(test, schemasDir)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if !result.Valid {
		t.Fatalf("negated schema validation failed: %+v", result.Errors)
	}
}
