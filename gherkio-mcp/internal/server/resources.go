package server

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ResourceIndex exposes catalog/flow/fixture/feature files to MCP clients.
type ResourceIndex struct {
	base string
}

// ResourceDescriptor matches MCP resource descriptions.
type ResourceDescriptor struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MimeType    string `json:"mimeType"`
	ETag        string `json:"etag,omitempty"`
	UpdatedAt   string `json:"updatedAt,omitempty"`
}

// ResourceContent returns full file contents.
type ResourceContent struct {
	URI       string `json:"uri"`
	MimeType  string `json:"mimeType"`
	Encoding  string `json:"encoding"`
	Text      string `json:"text"`
	ETag      string `json:"etag,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

// NewResourceIndex prepares a scanner rooted at base.
func NewResourceIndex(base string) *ResourceIndex {
	return &ResourceIndex{base: base}
}

var resourceRoots = []struct {
	name        string
	label       string
	description string
}{
	{name: "envs", label: "Environment", description: "Environment configuration"},
	{name: "apis", label: "API Catalog", description: "API catalog"},
	{name: "flows", label: "Flow", description: "Reusable flow macro"},
	{name: "fixtures", label: "Fixture", description: "Fixture payload"},
	{name: "features", label: "Feature", description: "Gherkin feature"},
}

// List discovers resources under known directories.
func (r *ResourceIndex) List(ctx context.Context) ([]ResourceDescriptor, error) {
	base := r.base
	if base == "" {
		return nil, fmt.Errorf("resource base not configured")
	}
	var out []ResourceDescriptor
	for _, root := range resourceRoots {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		dir := filepath.Join(base, root.name)
		info, err := os.Stat(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("stat %s: %w", dir, err)
		}
		if !info.IsDir() {
			continue
		}
		err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				return nil
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			rel, err := filepath.Rel(base, path)
			if err != nil {
				return err
			}
			stat, err := d.Info()
			if err != nil {
				return err
			}
			etag := checksum(stat)
			uri := "gherkio://" + filepath.ToSlash(rel)
			out = append(out, ResourceDescriptor{
				URI:         uri,
				Name:        filepath.ToSlash(rel),
				Description: fmt.Sprintf("%s %s", root.description, filepath.Base(path)),
				MimeType:    mimeTypeFor(path),
				ETag:        etag,
				UpdatedAt:   stat.ModTime().UTC().Format(time.RFC3339Nano),
			})
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out, nil
}

// Read loads a resource by URI.
func (r *ResourceIndex) Read(ctx context.Context, uri string) (ResourceContent, error) {
	rel, err := r.resolve(uri)
	if err != nil {
		return ResourceContent{}, err
	}
	full := filepath.Join(r.base, rel)
	select {
	case <-ctx.Done():
		return ResourceContent{}, ctx.Err()
	default:
	}
	data, err := os.ReadFile(full)
	if err != nil {
		if os.IsNotExist(err) {
			return ResourceContent{}, mcp.ResourceNotFoundError(uri)
		}
		return ResourceContent{}, fmt.Errorf("read %s: %w", uri, err)
	}
	info, err := os.Stat(full)
	if err != nil {
		return ResourceContent{}, fmt.Errorf("stat %s: %w", uri, err)
	}
	return ResourceContent{
		URI:       uri,
		MimeType:  mimeTypeFor(full),
		Encoding:  "utf-8",
		Text:      string(data),
		ETag:      checksum(info),
		UpdatedAt: info.ModTime().UTC().Format(time.RFC3339Nano),
	}, nil
}

func (r *ResourceIndex) resolve(uri string) (string, error) {
	if !strings.HasPrefix(uri, "gherkio://") {
		return "", fmt.Errorf("invalid uri scheme: %s", uri)
	}
	rel := strings.TrimPrefix(uri, "gherkio://")
	if strings.TrimSpace(rel) == "" {
		return "", fmt.Errorf("uri is empty")
	}
	relPath := filepath.Clean(filepath.FromSlash(rel))
	if relPath == ".." || strings.HasPrefix(relPath, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("uri escapes workspace: %s", uri)
	}
	return relPath, nil
}

func checksum(info fs.FileInfo) string {
	h := sha1.New()
	fmt.Fprintf(h, "%d:%d", info.ModTime().UnixNano(), info.Size())
	return hex.EncodeToString(h.Sum(nil))
}

func mimeTypeFor(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		return "application/yaml"
	case ".json":
		return "application/json"
	case ".feature":
		return "text/x.gherkin"
	case ".md":
		return "text/markdown"
	default:
		return "text/plain"
	}
}
