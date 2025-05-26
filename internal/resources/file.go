package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/geiserx/genieacs-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterFile wires the /file/{name} resource into the server.
func RegisterFile(s *server.MCPServer, acs *client.ACSClient) {
	tpl := mcp.NewResourceTemplate(
		"genieacs://file/{name}",
		"Firmware / file metadata",
		mcp.WithTemplateDescription("Metadata for a file stored in GenieACS"),
		mcp.WithTemplateMIMEType("application/json"),
	)

	s.AddResourceTemplate(tpl, func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		name := strings.TrimPrefix(req.Params.URI, "genieacs://file/")
		if name == "" {
			return nil, fmt.Errorf("missing file name")
		}

		body, err := acs.GetFileByName(name)
		if err != nil {
			return nil, err
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     string(body),
			},
		}, nil
	})
}
