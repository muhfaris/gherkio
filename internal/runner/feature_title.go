package runner

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	featureTitleCache = map[string]string{}
	featureTitleMu    sync.RWMutex
)

func featureTitleFor(uri string) string {
	if uri == "" {
		return ""
	}
	featureTitleMu.RLock()
	if title, ok := featureTitleCache[uri]; ok {
		featureTitleMu.RUnlock()
		return title
	}
	featureTitleMu.RUnlock()

	name := readFeatureTitle(uri)
	if name == "" {
		name = filepath.Base(uri)
	}
	featureTitleMu.Lock()
	featureTitleCache[uri] = name
	featureTitleMu.Unlock()
	return name
}

func readFeatureTitle(uri string) string {
	f, err := os.Open(uri)
	if err != nil {
		return ""
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "Feature:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Feature:"))
		}
	}
	return ""
}
