package resources

import (
	"context"

	"github.com/geiserx/genieacs-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterPresets wires genieacs://presets/list into the server.
func RegisterPresets(s *server.MCPServer, acs *client.ACSClient) {
	res := mcp.NewResource(
		"genieacs://presets/list",
		"Preset configuration list",
		mcp.WithResourceDescription(
			"Returns a JSON array of all presets configured in GenieACS. "+
				"Each preset contains a name, weight (priority), precondition (a MongoDB-style filter "+
				"that determines which devices the preset applies to), and configurations (an array of "+
				"value assignments, provision script references, or object add/delete operations). "+
				"Use this resource to inspect existing automation rules before creating or modifying "+
				"presets with the manage_preset tool. "+
				"Returns an empty array if no presets are configured.",
		),
		mcp.WithMIMEType("application/json"),
	)

	s.AddResource(res, func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		body, err := acs.ListPresets()
		if err != nil {
			return nil, err
		}
		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      "genieacs://presets/list",
				MIMEType: "application/json",
				Text:     string(body),
			},
		}, nil
	})
}
