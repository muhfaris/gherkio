package runner

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type FixturePayload struct {
	Body      []byte
	Multipart *MultipartPayload
}

type multipartFixtureDoc struct {
	Type  string                    `yaml:"type"`
	Parts []multipartFixtureDocPart `yaml:"parts"`
}

type multipartFixtureDocPart struct {
	Name        string `yaml:"name"`
	Value       string `yaml:"value,omitempty"`
	File        string `yaml:"file,omitempty"`
	Filename    string `yaml:"filename,omitempty"`
	ContentType string `yaml:"contentType,omitempty"`
}

func LoadFixtureFile(path string, store map[string]any) (FixturePayload, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return FixturePayload{}, err
	}

	ctx := map[string]any{"store": store}
	if vars, ok := store["vars"]; ok {
		ctx["vars"] = vars
	}
	rendered, err := execTemplate(string(data), ctx)
	if err != nil {
		return FixturePayload{}, err
	}

	// Try to decode as multipart fixture
	var doc multipartFixtureDoc
	if err := yaml.Unmarshal([]byte(rendered), &doc); err == nil && strings.EqualFold(strings.TrimSpace(doc.Type), "multipart") {
		if len(doc.Parts) == 0 {
			return FixturePayload{}, errors.New("multipart fixture has no parts")
		}
		baseDir := filepath.Dir(path)
		payload := &MultipartPayload{}
		for _, p := range doc.Parts {
			name := strings.TrimSpace(p.Name)
			if name == "" {
				return FixturePayload{}, errors.New("multipart fixture part missing name")
			}
			part := MultipartPart{
				Name:        name,
				ContentType: strings.TrimSpace(p.ContentType),
				Filename:    strings.TrimSpace(p.Filename),
			}
			if strings.TrimSpace(p.File) != "" {
				resolved := filepath.Clean(filepath.Join(baseDir, p.File))
				part.FilePath = resolved
				if part.Filename == "" {
					part.Filename = filepath.Base(resolved)
				}
			} else {
				part.Value = p.Value
			}
			payload.Parts = append(payload.Parts, part)
		}
		return FixturePayload{Multipart: payload}, nil
	}

	return FixturePayload{Body: []byte(rendered)}, nil
}

func ApplyFixture(req *Request, payload FixturePayload) {
	if payload.Multipart != nil {
		req.Multipart = payload.Multipart
		req.Body = nil
		if req.Headers != nil {
			delete(req.Headers, "Content-Type")
		}
		return
	}

	req.Multipart = nil
	req.Body = payload.Body
	if len(payload.Body) > 0 {
		if req.Headers == nil {
			req.Headers = map[string]string{}
		}
		if _, ok := req.Headers["Content-Type"]; !ok {
			req.Headers["Content-Type"] = "application/json"
		}
	}
}
