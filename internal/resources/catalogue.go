// Package resources registers the read-only MCP resources: the device list and
// single devices, files, tasks, faults, presets and provisions, each read from
// GenieACS through the client package.
package resources

import (
	"context"

	"github.com/geiserx/genieacs-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterCatalogue wires genieacs://devices/list into the server.
// The limit parameter controls the maximum number of devices returned
// and is configurable via the DEVICE_LIMIT environment variable.
func RegisterCatalogue(s *server.MCPServer, acs *client.ACSClient, limit int) {
	res := mcp.NewResource(
		"genieacs://devices/list",
		"Device summary list",
		mcp.WithResourceDescription(
			"Returns a lightweight JSON array of the CPE devices known to GenieACS (up to the configured "+
				"DEVICE_LIMIT, default 500). Each entry has the device _id, _lastInform, _tags and _deviceId "+
				"(_Manufacturer, _ProductClass, _SerialNumber). Use this resource as the starting point to discover "+
				"device IDs before calling tools or fetching individual device details with genieacs://device/{id}. "+
				"For more targeted queries, use the search_devices tool with MongoDB-style filters. "+
				"The response is an array of JSON objects. Returns an empty array if no devices are registered.",
		),
		mcp.WithMIMEType("application/json"),
	)

	s.AddResource(res, func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		body, err := acs.ListDeviceSummaries(ctx, limit)
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
