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
		"Queued and faulted tasks for a device",
		mcp.WithTemplateDescription(
			"Returns a JSON array of the tasks GenieACS still holds for a CPE device: the ones waiting to run "+
				"and the ones that faulted. GenieACS deletes a task once the device has run it, so a task missing "+
				"from this list has completed or was deleted. Each task has its name (e.g. reboot, download, "+
				"getParameterValues), timestamp and, if it faulted, a fault; details are in genieacs://faults/{id}. "+
				"The {id} parameter is the device's _id field. Returns an empty array if the device has no tasks.",
		),
		mcp.WithTemplateMIMEType("application/json"),
	)

	s.AddResourceTemplate(tpl, func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		id := strings.TrimPrefix(req.Params.URI, "genieacs://tasks/")
		if id == "" {
			return nil, fmt.Errorf("missing device id")
		}

		body, err := acs.GetTasksForDevice(ctx, id)
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
