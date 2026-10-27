package tools

import (
	"context"

	"github.com/geiserx/genieacs-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewRefreshParameter returns the refresh_parameter tool.
func NewRefreshParameter(acs *client.ACSClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := newTool("refresh_parameter", hints("Refresh parameter from device", false, false, true),
		mcp.WithDescription(
			"Ask a CPE device for the current value of a TR-069 parameter, or of every parameter under an object path, "+
				"and store it in GenieACS. Use it when get_parameter's cached value may be stale. It changes nothing on the device. "+
				"GenieACS queues a getParameterValues task and sends a connection request at once: status \"done\" means the "+
				"fresh value is already stored, so read it with get_parameter; \"queued\" means it arrives at the device's next inform. "+
				"Example: refresh_parameter(device_id=\"202BC1-BM632w-000001\", parameter=\"Device.DeviceInfo.SoftwareVersion\").",
		),
		deviceIDParam(),
		mcp.WithString("parameter",
			mcp.Required(),
			mcp.MinLength(1),
			mcp.Description("Full dotted TR-069 path in the device's data model. Examples: \"Device.DeviceInfo.SoftwareVersion\", "+
				"\"InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSID\", or an object such as \"Device.WiFi.\" for everything under it."),
		),
		mcp.WithOutputSchema[TaskOutput](),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		deviceID, err := req.RequireString("device_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		param, err := req.RequireString("parameter")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		res, err := acs.RefreshParameter(ctx, deviceID, param)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return taskResult(deviceID, "Refresh of "+param, res), nil
	}
	return tool, handler
}
