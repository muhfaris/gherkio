package report

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
)

var timestampDirRegex = regexp.MustCompile(`^\d{8}_\d{6}$`)

// EnforceRetention scans the reports directory, discovers all timestamped directories (YYYYMMDD_HHMMSS),
// and prunes the oldest runs if the total directory count exceeds the configured retention limit.
func EnforceRetention(projectDir string, customPath string, retentionLimit int) error {
	if retentionLimit <= 0 {
		return nil // Invalid or unlimited retention
	}

	basePath := filepath.Join(projectDir, ".gherkio", "reports")
	if customPath != "" {
		basePath = customPath
	}

	entries, err := os.ReadDir(basePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Nothing to prune if path doesn't exist yet
		}
		return fmt.Errorf("failed to read reports directory for retention cleanup: %w", err)
	}

	var timestampDirs []string
	for _, entry := range entries {
		if entry.IsDir() && timestampDirRegex.MatchString(entry.Name()) {
			timestampDirs = append(timestampDirs, entry.Name())
		}
	}

	// Sort directories alphabetically (which naturally matches chronological order due to YYYYMMDD_HHMMSS format)
	sort.Strings(timestampDirs)

	// If count exceeds limit, delete oldest entries (earliest in the sorted list)
	if len(timestampDirs) > retentionLimit {
		excessCount := len(timestampDirs) - retentionLimit
		for i := 0; i < excessCount; i++ {
			targetDir := filepath.Join(basePath, timestampDirs[i])
			if err := os.RemoveAll(targetDir); err != nil {
				// Log the error to stderr but don't halt test execution
				fmt.Fprintf(os.Stderr, "Warning: failed to prune old report directory %s: %v\n", targetDir, err)
			}
		}
	}

	return nil
}
