package runner

// GetCanonicalPaths returns the base paths supported for assertions and variables.
// Used by the JSON schema generator to provide editor autocomplete.
func GetCanonicalPaths() []string {
	return []string{"body", "headers", "jwt"}
}

// GetCollectionFunctions returns the collection matchers supported for array validations.
// Used by the JSON schema generator.
func GetCollectionFunctions() []string {
	return []string{"count", "all"}
}

// GetBackoffStrategies returns the supported retry backoff strategies.
// Used by the JSON schema generator and documentation.
func GetBackoffStrategies() []string {
	return []string{"constant", "linear", "exponential"}
}

// GetStepRoles returns the supported step lifecycle roles.
// Used by the JSON schema generator and documentation.
func GetStepRoles() []string {
	return []string{"setup", "steps", "teardown"}
}

// GetArgMatchers returns matchers that require an additional argument
// (e.g. "contains <value>", "regex <pattern>"). These matchers are listed
// as bare keywords in GetAvailableMatchers() for schema autocomplete,
// but their full form requires an argument to be valid.
func GetArgMatchers() []string {
	return []string{"contains", "startsWith", "endsWith", "regex"}
}
