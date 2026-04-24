package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/geiserx/genieacs-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterTasks wires genieacs://tasks/{id} into the server.
func RegisterTasks(s *server.MCPServer, acs *client.ACSClient) {
	tpl := mcp.NewResourceTemplate(
		"genieacs://tasks/{id}",
		"Pending and historical tasks for a device",
		mcp.WithTemplateDescription(
			"Returns a JSON array of all pending and completed tasks for a specific CPE device. "+
				"Each task includes its type (e.g. reboot, download, refreshObject), status, creation timestamp, "+
				"and any fault information if the task failed. Use this resource to check the result of a "+
				"previously issued tool action (reboot_device, download_firmware, refresh_parameter). "+
				"The {id} parameter is the device's _id field. Returns an empty array if the device has no tasks.",
		),
		mcp.WithTemplateMIMEType("application/json"),
	)

	s.AddResourceTemplate(tpl, func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		id := strings.TrimPrefix(req.Params.URI, "genieacs://tasks/")
		if id == "" {
			return nil, fmt.Errorf("missing device id")
		}

		body, err := acs.GetTasksForDevice(id)
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
