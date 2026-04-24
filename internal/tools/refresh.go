// internal/tools/refresh.go
package tools

import (
	"context"
	"fmt"

	"github.com/geiserx/genieacs-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewRefreshParameter defines the tool and its handler.
func NewRefreshParameter(acs *client.ACSClient) (mcp.Tool, server.ToolHandlerFunc) {

	tool := mcp.NewTool("refresh_parameter",
		mcp.WithDescription(
			"Request a CPE device to report the current value of a specific TR-069 parameter via GenieACS. "+
				"Use this tool when you need an up-to-date reading of a device parameter — for example, "+
				"checking the current firmware version, uptime, WiFi SSID, or any CWMP data model path. "+
				"The refreshed value is stored in the GenieACS device document and can be read afterwards "+
				"via genieacs://device/{id}. "+
				"Returns the raw JSON response from the ACS confirming the refresh task was queued. "+
				"Limitations: the refresh is asynchronous — the value is updated when the CPE next contacts the ACS. "+
				"Only one parameter path can be refreshed per call; use the full dotted TR-069 object path. "+
				"Example: refresh_parameter(device_id=\"00236A-SmartRG585-SMRT00236a42\", parameter=\"Device.DeviceInfo.SoftwareVersion\").",
		),
		mcp.WithString("device_id",
			mcp.Required(),
			mcp.Description(
				"The exact GenieACS device identifier (_id field). "+
					"Typically in the format OUI-ProductClass-SerialNumber. "+
					"Obtain valid IDs from the genieacs://devices/list resource.",
			),
		),
		mcp.WithString("parameter",
			mcp.Required(),
			mcp.Description(
				"Full TR-069 dotted parameter path to refresh. "+
					"Examples: \"Device.DeviceInfo.SoftwareVersion\", \"Device.DeviceInfo.SerialNumber\", "+
					"\"InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSID\". "+
					"Must be a valid path in the device's CWMP data model.",
			),
		),
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

		resp, err := acs.RefreshParameter(deviceID, param)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(
			fmt.Sprintf("Refresh task queued/processed. Raw ACS response: %s", string(resp)),
		), nil
	}

	return tool, handler
}
