package loader

import (
	"io/fs"
	"path/filepath"
	"strings"
)

// FindFeatures walks dir and returns all .feature files (recursive).
func FindFeatures(dir string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		name := strings.ToLower(d.Name())
		if strings.HasSuffix(name, ".feature") {
			out = append(out, path)
		}
		return nil
	})
	return out, err
}
