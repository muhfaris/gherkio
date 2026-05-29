# Adding a Custom Matcher

Gherkio's assertion engine is designed to be easily extensible. All matchers live under `internal/runner/matchers.go`.

---

## 🛠️ Step-by-Step Walkthrough

Follow these three simple steps to implement a custom matcher (for example, a `between` matcher to check if a numeric response is inside bounds):

### Step 1: Register the Matcher Keyword

Open [internal/runner/matchers.go](file:///home/muhfaris/Documents/projects/sourcecode/src/github.com/muhfaris/gherkio/internal/runner/matchers.go) and add the keyword to the slice of known matchers inside `GetAvailableMatchers()`:

```go
func GetAvailableMatchers() []string {
	return []string{
		"exists", "not exists", "uuid", "email", // ...
		"between", // <-- Register your matcher keyword here
	}
}
```

### Step 2: Implement the Evaluation Logic

Add the evaluation logic inside `evaluateAssertion` or helper dispatcher functions in the same file. Parse arguments if your matcher takes inputs (e.g. `between 1 10`):

```go
if strings.HasPrefix(expectedStr, "between ") {
	args := strings.TrimPrefix(expectedStr, "between ")
	parts := strings.Fields(args)
	if len(parts) != 2 {
		return AssertionResult{Passed: false, Reason: "between requires 2 arguments (min, max)"}
	}
	
	// Parse min and max as float64, compare against actual numeric value, and return:
	// return AssertionResult{Passed: true}
}
```

### Step 3: Add Unit Tests

Open [internal/runner/matchers_test.go](file:///home/muhfaris/Documents/projects/sourcecode/src/github.com/muhfaris/gherkio/internal/runner/matchers_test.go) and append test cases validating both passing and failing assertions:

```go
func TestMatchers(t *testing.T) {
	// Add test cases for your new matcher:
	// AssertPass(t, "between 1 10", 5)
	// AssertFail(t, "between 1 10", 12)
}
```
 Ensure your tests pass perfectly by running `go test -v ./internal/runner/`.
