package server

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server wraps the MCP SDK server setup for gherkio.
type Server struct {
	repoRoot     string
	resourcesDir string
	index        *ResourceIndex
}

// New constructs a server instance rooted at repoRoot.
func New(repoRoot, resourcesDir string) *Server {
	return &Server{
		repoRoot:     repoRoot,
		resourcesDir: resourcesDir,
		index:        NewResourceIndex(resourcesDir),
	}
}

// Run starts the MCP server over stdio using the official MCP Go SDK.
func (s *Server) Run(ctx context.Context) error {
	impl := &mcp.Implementation{Name: "gherkio-mcp", Version: "0.1.0"}
	mcpServer := mcp.NewServer(impl, nil)

	if err := s.registerResources(ctx, mcpServer); err != nil {
		return fmt.Errorf("register resources: %w", err)
	}
	if err := registerTools(s, mcpServer); err != nil {
		return fmt.Errorf("register tools: %w", err)
	}

	return mcpServer.Run(ctx, &mcp.StdioTransport{})
}

func (s *Server) registerResources(ctx context.Context, mcpServer *mcp.Server) error {
	descriptors, err := s.index.List(ctx)
	if err != nil {
		return err
	}
	for _, descriptor := range descriptors {
		if err := s.attachResource(mcpServer, descriptor); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) attachResource(mcpServer *mcp.Server, desc ResourceDescriptor) error {
	if _, err := url.Parse(desc.URI); err != nil {
		return fmt.Errorf("invalid resource uri %q: %w", desc.URI, err)
	}
	title := filepath.Base(desc.Name)
	if title == "" || title == "." {
		title = desc.Name
	}
	resource := &mcp.Resource{
		URI:         desc.URI,
		Name:        desc.Name,
		Title:       title,
		Description: desc.Description,
		MIMEType:    desc.MimeType,
	}
	mcpServer.AddResource(resource, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		content, err := s.index.Read(ctx, desc.URI)
		if err != nil {
			return nil, err
		}
		rc := &mcp.ResourceContents{
			URI:      desc.URI,
			MIMEType: content.MimeType,
			Text:     content.Text,
		}
		return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{rc}}, nil
	})
	return nil
}
