package resources

import (
	"context"

	"github.com/geiserx/genieacs-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterProvisions wires genieacs://provisions/list into the server.
func RegisterProvisions(s *server.MCPServer, acs *client.ACSClient) {
	res := mcp.NewResource(
		"genieacs://provisions/list",
		"Provision script list",
		mcp.WithResourceDescription(
			"Returns a JSON array of all provision scripts stored in GenieACS. "+
				"Each entry contains the script name and its JavaScript source code. "+
				"Provision scripts run server-side during CPE inform sessions and use the GenieACS "+
				"sandbox API (declare, commit, ext, log) to dynamically configure devices. "+
				"Use this resource to review existing scripts before creating or modifying "+
				"provisions with the manage_provision tool, or to check which scripts are "+
				"referenced by presets (via genieacs://presets/list). "+
				"Returns an empty array if no provision scripts exist.",
		),
		mcp.WithMIMEType("application/json"),
	)

	s.AddResource(res, func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		body, err := acs.ListProvisions()
		if err != nil {
			return nil, err
		}
		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      "genieacs://provisions/list",
				MIMEType: "application/json",
				Text:     string(body),
			},
		}, nil
	})
}
