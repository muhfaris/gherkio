package cmd

import (
	"testing"
	"time"
)

func TestParseRequestDelay(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Duration
		wantErr bool
	}{
		{name: "empty disables delay", input: "", want: 0},
		{name: "bare number defaults to milliseconds", input: "1000", want: time.Second},
		{name: "explicit milliseconds", input: "500ms", want: 500 * time.Millisecond},
		{name: "explicit seconds", input: "1s", want: time.Second},
		{name: "negative", input: "-1ms", wantErr: true},
		{name: "invalid", input: "later", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRequestDelay(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseRequestDelay(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("parseRequestDelay(%q) = %s, want %s", tt.input, got, tt.want)
			}
		})
	}
}

func TestRequestDelayForTarget(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		isDirectory bool
		want        time.Duration
	}{
		{name: "directory default", isDirectory: true, want: 100 * time.Millisecond},
		{name: "single file default", isDirectory: false, want: 50 * time.Millisecond},
		{name: "explicit override", input: "250", isDirectory: true, want: 250 * time.Millisecond},
		{name: "explicit zero disables directory default", input: "0", isDirectory: true, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := requestDelayForTarget(tt.input, tt.isDirectory)
			if err != nil {
				t.Fatalf("requestDelayForTarget(%q, %v): %v", tt.input, tt.isDirectory, err)
			}
			if got != tt.want {
				t.Errorf("requestDelayForTarget(%q, %v) = %s, want %s", tt.input, tt.isDirectory, got, tt.want)
			}
		})
	}
}
