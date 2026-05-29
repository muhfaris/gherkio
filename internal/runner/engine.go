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
	return []string{"contains", "startsWith", "endsWith", "regex", "gt", "gte", "lt", "lte"}
}

// MatcherInfo holds metadata about a single assertion matcher.
type MatcherInfo struct {
	Name        string
	Description string
	HasArg      bool // Whether the matcher requires an argument (e.g. "contains <value>")
}

// GetMatchersInfo returns metadata for all assertion matchers.
// This is the single source of truth — used by MCP resources and documentation.
func GetMatchersInfo() []MatcherInfo {
	matchers := GetAvailableMatchers()
	argMatchers := GetArgMatchers()
	argSet := make(map[string]bool, len(argMatchers))
	for _, m := range argMatchers {
		argSet[m] = true
	}

	descriptions := map[string]string{
		"exists":     "Field must exist (any value)",
		"not exists": "Field must NOT be present (negative assertion)",
		"uuid":       "Valid UUID v4 format",
		"email":      "Valid email format",
		"datetime":   "Valid RFC3339 / ISO8601 datetime",
		"uri":        "Valid URI format",
		"string":     "Value is a string type",
		"number":     "Value is a numeric type",
		"boolean":    "Value is a boolean type",
		"array":      "Value is an array type",
		"object":     "Value is an object type",
		"null":       "Value is null",
		"true":       "Value is boolean true",
		"false":      "Value is boolean false",
		"contains":   "String contains substring (case-sensitive)",
		"startsWith": "String starts with prefix",
		"endsWith":   "String ends with suffix",
		"regex":      "String matches regex pattern",
		"gt":         "Value is greater than (numeric)",
		"gte":        "Value is greater than or equal to (numeric)",
		"lt":         "Value is less than (numeric)",
		"lte":        "Value is less than or equal to (numeric)",
		"empty":      "String, array, or object is empty",
		"ipv4":       "Valid IPv4 address format",
		"ipv6":       "Valid IPv6 address format",
		"base64":     "Valid base64 encoded string",
		"mac":        "Valid MAC address format (e.g. aa:bb:cc:dd:ee:ff)",
	}

	var info []MatcherInfo
	for _, m := range matchers {
		info = append(info, MatcherInfo{
			Name:        m,
			Description: descriptions[m],
			HasArg:      argSet[m],
		})
	}
	return info
}

// VariableInfo holds metadata about a built-in generator variable.
type VariableInfo struct {
	Name        string
	Description string
	Example     string
}

// GetVariableInfo returns metadata for all built-in generator variables.
// This is the single source of truth — used by MCP resources and documentation.
func GetVariableInfo() []VariableInfo {
	return []VariableInfo{
		{Name: "$uuid", Description: "UUID v4 string", Example: "a1b2c3d4-e5f6-4789-abcd-ef1234567890"},
		{Name: "$ulid", Description: "ULID (timestamp + random)", Example: "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
		{Name: "$randomInt", Description: "Random integer 0-999999; use ${randomInt(min,max)} for custom range", Example: "74291"},
		{Name: "$randomEmail", Description: "Random email at @example.com", Example: "user_123456@example.com"},
		{Name: "$randomPhone", Description: "Random Indonesian-format phone number; use ${randomPhone(ISO)} or ${randomPhone(prefix)} for global formats", Example: "+6281234567890"},
		{Name: "$accounts.<name>.<field>", Description: "Access specific account fields from credentials", Example: "$accounts.alpha.username"},
		{Name: "$timestamp", Description: "Current Unix epoch timestamp in seconds", Example: "1716942900"},
		{Name: "$timestampMs", Description: "Current Unix epoch timestamp in milliseconds", Example: "1716942900123"},
		{Name: "${dateNow(format)}", Description: "Get current date/time formatted using custom Go layout, e.g. '2006-01-02' (default format: '2006-01-02 15:04:05')", Example: "${dateNow(\"2006-01-02\")}"},
		{Name: "${dateOffset(duration,format)}", Description: "Calculates current date/time with a duration offset and custom layout formatting", Example: "${dateOffset(\"+14d\",\"2006-01-02\")}"},
		{Name: "${base64(string)}", Description: "Encodes string to Base64 standard format", Example: "${base64(\"hello\")}"},
		{Name: "${base64Decode(encoded)}", Description: "Decodes Base64 string back to plaintext", Example: "${base64Decode(\"aGVsbG8=\")}"},
		{Name: "${urlencode(string)}", Description: "Encodes string for safe URL query inclusion", Example: "${urlencode(\"hello world\")}"},
		{Name: "${urldecode(encoded)}", Description: "Decodes URL-encoded string back to plaintext", Example: "${urldecode(\"hello+world\")}"},
		{Name: "${hash(algo,data)}", Description: "Generates a hex-encoded hash of the data string using the specified algorithm (md5, sha1, sha256)", Example: "${hash(\"sha256\",\"secret\")}"},
		{Name: "${hmac(algo,key,message)}", Description: "Generates a hex-encoded HMAC hash using algorithm (md5, sha1, sha256)", Example: "${hmac(\"sha256\",\"my-key\",\"message\")}"},
		{Name: "${randomString(length,charset)}", Description: "Generates random string of length with character set (alpha, numeric, alphanumeric)", Example: "${randomString(10,\"alphanumeric\")}"},
		{Name: "${toUpper(string)}", Description: "Converts a string value to uppercase", Example: "${toUpper(\"hello\")}"},
		{Name: "${toLower(string)}", Description: "Converts a string value to lowercase", Example: "${toLower(\"HELLO\")}"},
		{Name: "${trim(string)}", Description: "Trims whitespace from both ends of a string value", Example: "${trim(\"  hello  \")}"},
	}
}

// PathInfo holds metadata about a canonical assertion or save path.
type PathInfo struct {
	Path        string
	Description string
	Usage       string
}

// GetPathInfo returns metadata for all canonical assertion/save paths.
// This is the single source of truth — used by MCP resources and documentation.
func GetPathInfo() []PathInfo {
	info := []PathInfo{
		{Path: "body.<field>", Description: "Response JSON body field", Usage: "body.token, body.data.0.name"},
		{Path: "headers.<name>", Description: "Response header", Usage: "headers.content-type"},
		{Path: "jwt.<claim>", Description: "Decoded JWT claim (auto-decoded from body.token or body.access_token)", Usage: "jwt.role, jwt.sub"},
		{Path: "status", Description: "HTTP status code", Usage: "status"},
		{Path: "schema", Description: "Validate full response body against a YAML schema", Usage: "schema: <name> or schema: not <name>"},
	}

	for _, fn := range GetCollectionFunctions() {
		if fn == "count" {
			info = append(info, PathInfo{
				Path:        "count(<path>): <N>",
				Description: "Assert array has exactly N items",
				Usage:       "count(body.items): 3",
			})
		} else {
			info = append(info, PathInfo{
				Path:        "all(<path>): <matcher>",
				Description: "Assert every element in array matches",
				Usage:       "all(body.items): uuid",
			})
		}
	}

	return info
}
