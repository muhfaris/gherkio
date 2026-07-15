package runner

import (
	"os"
	"testing"
)

func TestScanSteps(t *testing.T) {
	content := `scenario: user test
# global metadata

setup:
  - request:
      method: POST
      url: /setup
    expect:
      status: 200

steps:
  - request:
      method: GET
      url: /users
    expect:
      status: 200

  - use: common/login.yaml

  - request:
      method: DELETE
      url: /users/1
    expect:
      status: 204

teardown:
  - request:
      method: POST
      url: /cleanup
`

	// Create a temp file
	tmpFile, err := os.CreateTemp("", "gherkio-test-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("failed to write content: %v", err)
	}
	tmpFile.Close()

	steps, err := ScanSteps(tmpFile.Name())
	if err != nil {
		t.Fatalf("unexpected error scanning steps: %v", err)
	}

	// We expect:
	// 1 setup step
	// 3 main steps (request, use, request)
	// 1 teardown step
	// Total = 5 steps
	if len(steps) != 5 {
		t.Fatalf("expected 5 steps, got %d", len(steps))
	}

	// Step 0: setup, index 0
	if steps[0].Section != "setup" || steps[0].Index != 0 {
		t.Errorf("step 0 mismatch: %+v", steps[0])
	}
	// Step 1: steps, index 0
	if steps[1].Section != "steps" || steps[1].Index != 0 {
		t.Errorf("step 1 mismatch: %+v", steps[1])
	}
	// Step 2: steps, index 1 (use)
	if steps[2].Section != "steps" || steps[2].Index != 1 {
		t.Errorf("step 2 mismatch: %+v", steps[2])
	}

	// Test LocateStep
	// Line 6 is "  - request:" under setup -> should be setup step 0
	loc, err := LocateStep(tmpFile.Name(), 6)
	if err != nil {
		t.Fatalf("failed to locate step at line 6: %v", err)
	}
	if loc.Section != "setup" || loc.Index != 0 {
		t.Errorf("expected setup step 0, got section %s index %d", loc.Section, loc.Index)
	}

	// Line 18 is "  - use: common/login.yaml" -> should be steps step 1
	loc, err = LocateStep(tmpFile.Name(), 18)
	if err != nil {
		t.Fatalf("failed to locate step at line 18: %v", err)
	}
	if loc.Section != "steps" || loc.Index != 1 {
		t.Errorf("expected steps step 1, got section %s index %d", loc.Section, loc.Index)
	}

	// Line 28 is teardown request -> should be teardown step 0
	loc, err = LocateStep(tmpFile.Name(), 28)
	if err != nil {
		t.Fatalf("failed to locate step at line 28: %v", err)
	}
	if loc.Section != "teardown" || loc.Index != 0 {
		t.Errorf("expected teardown step 0, got section %s index %d", loc.Section, loc.Index)
	}
}

func TestScanSteps_NamedSteps(t *testing.T) {
	content := `scenario: named steps test
steps:
  - use: auth/login.yaml
    with:
      username: $username

  - name: Create Ticket
    request:
      method: POST
      url: /v1/tickets
    expect:
      status: 201

  - name: Push Notification
    request:
      method: POST
      url: /v1/notifications
    expect:
      status: 200
`

	tmpFile, err := os.CreateTemp("", "gherkio-named-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("failed to write content: %v", err)
	}
	tmpFile.Close()

	steps, err := ScanSteps(tmpFile.Name())
	if err != nil {
		t.Fatalf("unexpected error scanning steps: %v", err)
	}

	// Expect 3 steps: 1 use + 2 named
	if len(steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(steps))
	}

	// Verify each step section and index
	if steps[0].Section != "steps" || steps[0].Index != 0 {
		t.Errorf("step 0 (use) mismatch: %+v", steps[0])
	}
	if steps[1].Section != "steps" || steps[1].Index != 1 {
		t.Errorf("step 1 (Create Ticket) mismatch: %+v", steps[1])
	}
	if steps[2].Section != "steps" || steps[2].Index != 2 {
		t.Errorf("step 2 (Push Notification) mismatch: %+v", steps[2])
	}

	// Test cursor on "- name: Create Ticket" line
	loc, err := LocateStep(tmpFile.Name(), 7)
	if err != nil {
		t.Fatalf("failed to locate step at line 7: %v", err)
	}
	if loc.Section != "steps" || loc.Index != 1 {
		t.Errorf("expected steps step 1 (Create Ticket), got section %s index %d", loc.Section, loc.Index)
	}

	// Test cursor on "- name: Push Notification" line
	loc, err = LocateStep(tmpFile.Name(), 14)
	if err != nil {
		t.Fatalf("failed to locate step at line 14: %v", err)
	}
	if loc.Section != "steps" || loc.Index != 2 {
		t.Errorf("expected steps step 2 (Push Notification), got section %s index %d", loc.Section, loc.Index)
	}

	// Test cursor on "- use: auth/login.yaml" line
	loc, err = LocateStep(tmpFile.Name(), 3)
	if err != nil {
		t.Fatalf("failed to locate step at line 3: %v", err)
	}
	if loc.Section != "steps" || loc.Index != 0 {
		t.Errorf("expected steps step 0 (use), got section %s index %d", loc.Section, loc.Index)
	}
}

func TestScanSteps_NestedLists(t *testing.T) {
	content := `scenario: nested lists test
steps:
  - name: Step One
    request:
      method: POST
      url: /v1/tickets
      body:
        members:
          - id: 1
          - id: 2
    retry:
      onStatus:
        - 400
        - 409
  - name: Step Two
    request:
      method: GET
      url: /v1/tickets/1
`

	tmpFile, err := os.CreateTemp("", "gherkio-nested-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("failed to write content: %v", err)
	}
	tmpFile.Close()

	steps, err := ScanSteps(tmpFile.Name())
	if err != nil {
		t.Fatalf("unexpected error scanning steps: %v", err)
	}

	// We expect exactly 2 steps, despite the nested lists
	if len(steps) != 2 {
		t.Fatalf("expected 2 steps, got %d. Steps: %+v", len(steps), steps)
	}

	if steps[0].Section != "steps" || steps[0].Index != 0 {
		t.Errorf("step 0 mismatch: %+v", steps[0])
	}
	if steps[1].Section != "steps" || steps[1].Index != 1 {
		t.Errorf("step 1 mismatch: %+v", steps[1])
	}
}
