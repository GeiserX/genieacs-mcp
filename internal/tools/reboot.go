// Package tools defines the MCP tools that act on GenieACS. Each New function
// returns a tool definition and the handler that calls the matching client
// method; the server registers both.
package tools

import (
	"context"

	"github.com/geiserx/genieacs-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewReboot returns the reboot_device tool.
func NewReboot(acs *client.ACSClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := newTool("reboot_device", hints("Reboot device", false, true, false),
		mcp.WithDescription(
			"Reboot a CPE device through GenieACS. Use it when a device needs a restart, for example after a "+
				"configuration change or to recover an unresponsive device. The device drops its service while it restarts. "+
				"GenieACS queues the task and sends a connection request at once: status \"done\" means the device ran it "+
				"in that session, \"queued\" means it runs at the device's next inform. A reboot does not confirm the device "+
				"came back; read Device.DeviceInfo.UpTime (or InternetGatewayDevice.DeviceInfo.UpTime) with refresh_parameter afterwards. "+
				"Example: reboot_device(device_id=\"202BC1-BM632w-000001\").",
		),
		deviceIDParam(),
		mcp.WithOutputSchema[TaskOutput](),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		devID, err := req.RequireString("device_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		res, err := acs.RebootDevice(ctx, devID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return taskResult(devID, "Reboot", res), nil
	}
	return tool, handler
}
