package report

import (
	"encoding/csv"
	"fmt"
	"os"
	"time"

	"github.com/muhfaris/gherkio/internal/fsutil"
	"github.com/muhfaris/gherkio/internal/runner"
)

type CSV struct{}

func NewCSV() *CSV { return &CSV{} }

func (c *CSV) Append(path, feature, scenario, status string, durMs int64) error {
	if err := fsutil.EnsureDir(dirOf(path)); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	if fi, _ := f.Stat(); fi.Size() == 0 {
		_ = w.Write([]string{"feature", "scenario", "status", "duration_ms", "timestamp"})
	}
	return w.Write([]string{feature, scenario, status, itoa(durMs), time.Now().UTC().Format(time.RFC3339)})
}

func (c *CSV) AppendSingle(path string, req runner.Request, resp runner.Response) error {
	return c.Append(path, "single", req.APIKey, statusFrom(resp.Status), 0)
}

func statusFrom(code int) string {
	if code >= 200 && code < 300 {
		return "PASSED"
	} else {
		return "FAILED"
	}
}

func dirOf(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' {
			return p[:i]
		}
	}
	return "."
}

func itoa(n int64) string { return fmt.Sprintf("%d", n) }
