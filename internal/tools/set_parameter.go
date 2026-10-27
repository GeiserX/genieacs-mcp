package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/geiserx/genieacs-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewSetParameter returns the set_parameter tool, which queues a
// setParameterValues task on a device and asks GenieACS to run it at once.
func NewSetParameter(acs *client.ACSClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := newTool("set_parameter", hints("Set device parameters", false, true, true),
		mcp.WithDescription(
			"Write one or more TR-069 parameter values on a CPE device, overwriting the current values: WiFi SSID or key, "+
				"periodic inform interval, management server URL, or any writable parameter. A wrong value can cut the device off "+
				"(a wrong WAN or ACS URL setting, for example). parameter_values is a JSON array of [path, value] or "+
				"[path, value, xsdType] tuples; xsd types include xsd:string, xsd:boolean, xsd:unsignedInt, xsd:int and xsd:dateTime, "+
				"and GenieACS infers the type when it is omitted. "+
				"Example: [[\"InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSID\",\"Lab-2\",\"xsd:string\"]]. "+
				"GenieACS queues the task and sends a connection request at once: status \"done\" means the device applied it, "+
				"\"queued\" means it runs at the next inform. A read-only or unknown parameter makes the device fault the task; "+
				"see genieacs://faults/{device_id}. Read the current value with get_parameter first.",
		),
		deviceIDParam(),
		mcp.WithString("parameter_values",
			mcp.Required(),
			mcp.MinLength(2),
			mcp.Description("JSON array of tuples, each [path, value] or [path, value, xsdType]. "+
				"Example: [[\"Device.WiFi.SSID.1.SSID\",\"NewName\",\"xsd:string\"],[\"Device.ManagementServer.PeriodicInformInterval\",300,\"xsd:unsignedInt\"]]"),
		),
		mcp.WithOutputSchema[TaskOutput](),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		devID, err := req.RequireString("device_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		pvStr, ok := stringOrJSON(req.GetArguments()["parameter_values"])
		if !ok {
			return mcp.NewToolResultError("required argument \"parameter_values\" not found"), nil
		}
		var tuples [][]any
		if err := json.Unmarshal([]byte(pvStr), &tuples); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("parameter_values is not valid JSON: it must be an array of [path, value] or [path, value, xsdType] arrays: %v", err)), nil
		}
		if err := checkTuples(tuples); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		res, err := acs.SetParameterValues(ctx, devID, json.RawMessage(pvStr))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return taskResult(devID, fmt.Sprintf("Setting %d parameter(s)", len(tuples)), res), nil
	}
	return tool, handler
}

// checkTuples validates setParameterValues tuples before they reach GenieACS.
func checkTuples(tuples [][]any) error {
	if len(tuples) == 0 {
		return fmt.Errorf("parameter_values is empty: give at least one [path, value] tuple")
	}
	for i, t := range tuples {
		if len(t) < 2 || len(t) > 3 {
			return fmt.Errorf("parameter_values[%d] has %d elements: each tuple is [path, value] or [path, value, xsdType]", i, len(t))
		}
		if p, ok := t[0].(string); !ok || p == "" {
			return fmt.Errorf("parameter_values[%d][0] must be the parameter path as a string", i)
		}
		switch t[1].(type) {
		case string, float64, bool:
		default:
			return fmt.Errorf("parameter_values[%d][1] must be a string, number or boolean", i)
		}
		if len(t) == 3 {
			if typ, ok := t[2].(string); !ok || typ == "" {
				return fmt.Errorf("parameter_values[%d][2] must be an xsd type such as \"xsd:string\"", i)
			}
		}
	}
	return nil
}
