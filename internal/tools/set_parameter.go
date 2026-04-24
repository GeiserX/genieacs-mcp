package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/geiserx/genieacs-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func NewSetParameter(acs *client.ACSClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("set_parameter",
		mcp.WithDescription(
			"Set one or more TR-069 parameter values on a CPE device through GenieACS. "+
				"Use this tool to change device configuration such as WiFi SSID, management server URL, "+
				"periodic inform interval, or any writable TR-069 parameter. "+
				"The parameter_values argument must be a JSON array of tuples, where each tuple is "+
				"[parameterPath, value] or [parameterPath, value, xsdType]. "+
				"Example: [[\"InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSID\",\"MySSID\",\"xsd:string\"]]. "+
				"Valid xsd types include xsd:string, xsd:boolean, xsd:unsignedInt, xsd:int, xsd:dateTime. "+
				"If the type is omitted, GenieACS infers it from the value. "+
				"The task is queued and a connection request is sent to the CPE for immediate execution. "+
				"Returns the raw ACS JSON response confirming the task. "+
				"Limitations: the CPE must be reachable for immediate execution; otherwise the task "+
				"remains queued until the next periodic inform. Not all parameters are writable — "+
				"the CPE will fault if you attempt to set a read-only parameter. "+
				"Use refresh_parameter or genieacs://device/{id} first to discover valid parameter paths.",
		),
		mcp.WithString("device_id",
			mcp.Required(),
			mcp.Description(
				"The exact GenieACS device identifier (_id field). "+
					"Typically in the format OUI-ProductClass-SerialNumber (e.g. \"00236A-SmartRG585-SMRT00236a42\"). "+
					"Obtain valid IDs from the genieacs://devices/list resource.",
			),
		),
		mcp.WithString("parameter_values",
			mcp.Required(),
			mcp.Description(
				"A JSON array of parameter tuples to set. Each tuple is either "+
					"[path, value] or [path, value, xsdType]. "+
					"Example: [[\"Device.WiFi.SSID.1.SSID\",\"NewName\",\"xsd:string\"],[\"Device.ManagementServer.PeriodicInformInterval\",300,\"xsd:unsignedInt\"]]",
			),
		),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		devID, err := req.RequireString("device_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		pvStr, err := req.RequireString("parameter_values")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		var pv json.RawMessage
		if err := json.Unmarshal([]byte(pvStr), &pv); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("parameter_values is not valid JSON: %v", err)), nil
		}

		resp, err := acs.SetParameterValues(devID, pv)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(
			fmt.Sprintf("setParameterValues task queued. Raw ACS response: %s", string(resp)),
		), nil
	}

	return tool, handler
}
