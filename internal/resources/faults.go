package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/geiserx/genieacs-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterFaults wires genieacs://faults/{id} into the server.
func RegisterFaults(s *server.MCPServer, acs *client.ACSClient) {
	tpl := mcp.NewResourceTemplate(
		"genieacs://faults/{id}",
		"Device fault records",
		mcp.WithTemplateDescription(
			"Returns a JSON array of fault records for a specific CPE device in GenieACS. "+
				"Faults are created when a queued task fails during execution — for example, when a "+
				"firmware download is rejected by the CPE, a parameter set targets a read-only value, "+
				"or the CPE returns an internal error during a CWMP session. "+
				"Each fault includes the fault code, message, detail, timestamp, retry count, and the "+
				"associated task information. The {id} parameter is the device's _id field. "+
				"Use this resource to diagnose why a previously issued tool action (reboot_device, "+
				"download_firmware, set_parameter) failed. "+
				"After identifying the fault, use retry_task to re-attempt the failed task, or "+
				"delete_task to cancel it. "+
				"Returns an empty array if the device has no faults.",
		),
		mcp.WithTemplateMIMEType("application/json"),
	)

	s.AddResourceTemplate(tpl, func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		id := strings.TrimPrefix(req.Params.URI, "genieacs://faults/")
		if id == "" {
			return nil, fmt.Errorf("missing device id")
		}

		body, err := acs.GetFaultsForDevice(id)
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
