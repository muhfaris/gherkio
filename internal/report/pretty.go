package report

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/muhfaris/gherkio/internal/runner"
)

type Pretty struct{ w io.Writer }

func NewPretty(w io.Writer) *Pretty { return &Pretty{w: w} }

func (p *Pretty) StartFeature(name string)           { fmt.Fprintf(p.w, "Feature: %s\n", name) }
func (p *Pretty) RecordScenario(name, status string) { fmt.Fprintf(p.w, "  %s: %s\n", name, status) }

func (p *Pretty) RecordSingle(req runner.Request, resp runner.Response) {
	fmt.Fprintf(p.w, "API: %s -> status %d\n", req.APIKey, resp.Status)
	if len(resp.Body) > 0 {
		var js map[string]any
		if json.Unmarshal(resp.Body, &js) == nil {
			b, _ := json.MarshalIndent(js, "", "  ")
			fmt.Fprintf(p.w, "%s\n", string(b))
		} else {
			fmt.Fprintf(p.w, "%s\n", string(resp.Body))
		}
	}
}
