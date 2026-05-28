package cmd

import (
	"bytes"
	"os"
	"testing"
)

func TestSchemaCmd_ListTypes(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Set flags
	schemaType = ""
	listTypes = true

	// Run command
	err := schemaCmd.RunE(schemaCmd, []string{})

	// Restore stdout
	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)

	if err != nil {
		t.Fatalf("schemaCmd.RunE() error = %v", err)
	}

	output := buf.String()

	// Verify output contains expected types
	expectedTypes := []string{"test", "config", "environment", "credentials", "schema-definition"}
	for _, typ := range expectedTypes {
		if !bytes.Contains([]byte(output), []byte(typ)) {
			t.Errorf("Expected output to contain type %q", typ)
		}
	}

	// Verify it mentions file patterns
	if !bytes.Contains([]byte(output), []byte(".gherkio")) {
		t.Error("Expected output to mention .gherkio file patterns")
	}
}

func TestPrintSchemaTypes_Output(t *testing.T) {
	err := printSchemaTypes()
	if err != nil {
		t.Fatalf("printSchemaTypes() error = %v", err)
	}
}
