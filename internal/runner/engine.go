package runner

// GetCanonicalPaths returns the base paths supported for assertions and variables.
// Used by the JSON schema generator to provide editor autocomplete.
func GetCanonicalPaths() []string {
	return []string{"body", "headers", "jwt"}
}
