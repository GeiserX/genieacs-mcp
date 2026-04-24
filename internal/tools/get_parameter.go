package tools

import (
	"context"
	"fmt"

	"github.com/geiserx/genieacs-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func NewGetParameter(acs *client.ACSClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("get_parameter",
		mcp.WithDescription(
			"Read specific TR-069 parameter values from the GenieACS cache for a device without "+
				"contacting the CPE. Use this tool to quickly retrieve known parameter values such as "+
				"firmware version, serial number, uptime, WiFi SSID, or IP addresses. "+
				"This reads the last-known values stored in the ACS database — the data may be stale "+
				"if the device has not informed recently. Use refresh_parameter first if you need a "+
				"guaranteed fresh value from the CPE. "+
				"The parameter_path can be a single parameter (e.g. \"Device.DeviceInfo.SoftwareVersion\") "+
				"or a comma-separated list for multiple parameters. "+
				"Returns the matching device document fields as JSON. "+
				"Limitations: only returns data that the ACS has previously collected. "+
				"If a parameter has never been read from the CPE, it will not appear in the response.",
		),
		mcp.WithString("device_id",
			mcp.Required(),
			mcp.Description(
				"The exact GenieACS device identifier (_id field). "+
					"Typically in the format OUI-ProductClass-SerialNumber (e.g. \"00236A-SmartRG585-SMRT00236a42\"). "+
					"Obtain valid IDs from the genieacs://devices/list resource.",
			),
		),
		mcp.WithString("parameter_path",
			mcp.Required(),
			mcp.Description(
				"Comma-separated TR-069 parameter paths to retrieve from the ACS cache. "+
					"Example: \"Device.DeviceInfo.SoftwareVersion\" or "+
					"\"Device.DeviceInfo.SoftwareVersion,Device.DeviceInfo.UpTime,Device.ManagementServer.PeriodicInformInterval\"",
			),
		),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		devID, err := req.RequireString("device_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		paramPath, err := req.RequireString("parameter_path")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		resp, err := acs.GetDeviceParameters(devID, paramPath)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(
			fmt.Sprintf("Cached parameter values: %s", string(resp)),
		), nil
	}

	return tool, handler
}
