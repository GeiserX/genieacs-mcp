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
		mcp.WithTemplateDescription(
			"Returns the full JSON document for a single CPE device as stored in GenieACS. "+
				"Contains all TR-069 parameters (Device.DeviceInfo.*, Device.ManagementServer.*, etc.), "+
				"tags, last inform timestamp, and task history. Use genieacs://devices/list first to discover "+
				"valid device IDs, then fetch individual devices with this resource. "+
				"The {id} parameter is the device's _id field (typically OUI-ProductClass-SerialNumber). "+
				"Response is a single JSON object. Returns a 404 error if the device ID does not exist.",
		),
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
