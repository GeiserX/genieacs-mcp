package resources

import (
	"context"

	"github.com/geiserx/genieacs-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterCatalogue wires genieacs://devices/list into the server.
func RegisterCatalogue(s *server.MCPServer, acs *client.ACSClient) {
	res := mcp.NewResource(
		"genieacs://devices/list",
		"Device summary list",
		mcp.WithResourceDescription("Lightweight list of devices and last inform time"),
		mcp.WithMIMEType("application/json"),
	)

	s.AddResource(res, func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		body, err := acs.ListDeviceSummaries(500) // limit hard-coded as before
		if err != nil {
			return nil, err
		}
		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      "genieacs://devices/list",
				MIMEType: "application/json",
				Text:     string(body),
			},
		}, nil
	})
}
