package runner

import (
	"testing"

	"github.com/muhfaris/gherkio/internal/model"
)

func TestParseUntil(t *testing.T) {
	// Create a mock TestFile
	testFile := &model.TestFile{
		Setup: []model.Step{
			{Request: model.Request{URL: "http://setup1"}},
			{Request: model.Request{URL: "http://setup2"}},
		},
		Steps: []model.Step{
			{Request: model.Request{URL: "http://step1"}},
			{Request: model.Request{URL: "http://step2"}},
			{Request: model.Request{URL: "http://step3"}},
		},
		Teardown: []model.Step{
			{Request: model.Request{URL: "http://teardown1"}},
		},
	}

	tests := []struct {
		name        string
		untilStr    string
		wantSection string
		wantIndex   int
		expectError bool
	}{
		{
			name:        "empty until string",
			untilStr:    "",
			wantSection: "",
			wantIndex:   -1,
			expectError: false,
		},
		{
			name:        "default section (steps) with index",
			untilStr:    "1",
			wantSection: "steps",
			wantIndex:   1,
			expectError: false,
		},
		{
			name:        "setup section with index",
			untilStr:    "setup:0",
			wantSection: "setup",
			wantIndex:   0,
			expectError: false,
		},
		{
			name:        "steps section with index",
			untilStr:    "steps:2",
			wantSection: "steps",
			wantIndex:   2,
			expectError: false,
		},
		{
			name:        "teardown section with index",
			untilStr:    "teardown:0",
			wantSection: "teardown",
			wantIndex:   0,
			expectError: false,
		},
		{
			name:        "case insensitivity of section",
			untilStr:    "SeTuP:1",
			wantSection: "setup",
			wantIndex:   1,
			expectError: false,
		},
		{
			name:        "whitespace trimming",
			untilStr:    "  steps  :  1  ",
			wantSection: "steps",
			wantIndex:   1,
			expectError: false,
		},
		{
			name:        "invalid index format",
			untilStr:    "steps:abc",
			expectError: true,
		},
		{
			name:        "negative index",
			untilStr:    "steps:-1",
			expectError: true,
		},
		{
			name:        "setup index out of bounds",
			untilStr:    "setup:2",
			expectError: true,
		},
		{
			name:        "steps index out of bounds",
			untilStr:    "steps:3",
			expectError: true,
		},
		{
			name:        "teardown index out of bounds",
			untilStr:    "teardown:1",
			expectError: true,
		},
		{
			name:        "invalid section name",
			untilStr:    "invalidsec:1",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sec, idx, err := parseUntil(tt.untilStr, testFile)
			if (err != nil) != tt.expectError {
				t.Errorf("parseUntil(%q) error = %v, expectError = %v", tt.untilStr, err, tt.expectError)
				return
			}
			if !tt.expectError {
				if sec != tt.wantSection {
					t.Errorf("parseUntil(%q) got section = %q, want %q", tt.untilStr, sec, tt.wantSection)
				}
				if idx != tt.wantIndex {
					t.Errorf("parseUntil(%q) got index = %d, want %d", tt.untilStr, idx, tt.wantIndex)
				}
			}
		})
	}
}
