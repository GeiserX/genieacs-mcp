package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/geiserx/genieacs-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterDevice wires the template + handler into the MCP server.
func RegisterDevice(s *server.MCPServer, acs *client.ACSClient) {
	tpl := mcp.NewResourceTemplate(
		"genieacs://device/{id}",
		"GenieACS device JSON",
		mcp.WithTemplateDescription("Raw device document as returned by NBI"),
		mcp.WithTemplateMIMEType("application/json"),
	)

	s.AddResourceTemplate(tpl, func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		id := strings.TrimPrefix(req.Params.URI, "genieacs://device/")
		if id == "" {
			return nil, fmt.Errorf("missing device id")
		}

		body, err := acs.GetDevice(id)
		if err != nil {
			return nil, err
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      fmt.Sprintf("genieacs://device/%s", id),
				MIMEType: "application/json",
				Text:     string(body),
			},
		}, nil
	})
}
