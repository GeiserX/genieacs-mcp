package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/geiserx/genieacs-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ParameterValue is one cached TR-069 parameter.
type ParameterValue struct {
	Value     any    `json:"value" jsonschema_description:"The last value the device reported."`
	Type      string `json:"type,omitempty" jsonschema_description:"Its xsd type, e.g. xsd:string."`
	Timestamp string `json:"timestamp,omitempty" jsonschema_description:"When GenieACS last read it from the device (ISO 8601)."`
}

// GetParameterOutput is the structured result of get_parameter.
type GetParameterOutput struct {
	DeviceID   string                    `json:"device_id"`
	Found      bool                      `json:"found" jsonschema_description:"false when no device has this id."`
	Parameters map[string]ParameterValue `json:"parameters" jsonschema_description:"Every cached leaf parameter at or under the requested paths, keyed by full path."`
	Missing    []string                  `json:"missing" jsonschema_description:"Requested paths with no cached value; refresh_parameter can fetch them."`
}

// NewGetParameter returns the get_parameter tool, which reads parameter values
// GenieACS has cached for a device, without contacting the CPE.
func NewGetParameter(acs *client.ACSClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := newTool("get_parameter", hints("Read cached device parameters", true, false, true),
		mcp.WithDescription(
			"Read TR-069 parameter values GenieACS has cached for a device, without contacting the device: firmware version, "+
				"serial number, uptime, WiFi SSID, IP addresses and so on. Values are what the device reported at its last "+
				"inform or refresh, with a timestamp each; call refresh_parameter first when you need a value from the device now. "+
				"Give one path or a comma-separated list; an object path such as \"Device.WiFi\" returns every cached leaf under it. "+
				"Returns found=false for an unknown device id, and lists paths with no cached value under missing.",
		),
		deviceIDParam(),
		mcp.WithString("parameter_path",
			mcp.Required(),
			mcp.MinLength(1),
			mcp.Description("Comma-separated TR-069 paths. Example: \"Device.DeviceInfo.SoftwareVersion\" or "+
				"\"InternetGatewayDevice.DeviceInfo.SoftwareVersion,InternetGatewayDevice.DeviceInfo.UpTime\"."),
		),
		mcp.WithOutputSchema[GetParameterOutput](),
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
		var paths []string
		for _, p := range strings.Split(paramPath, ",") {
			if p = strings.Trim(strings.TrimSpace(p), "."); p != "" {
				paths = append(paths, p)
			}
		}
		if len(paths) == 0 {
			return mcp.NewToolResultError("parameter_path has no paths"), nil
		}

		resp, err := acs.GetDeviceParameters(ctx, devID, strings.Join(paths, ","))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		var docs []map[string]any
		if err := json.Unmarshal(resp, &docs); err != nil {
			return mcp.NewToolResultError("GenieACS returned something other than a device list"), nil
		}

		out := GetParameterOutput{DeviceID: devID, Parameters: map[string]ParameterValue{}, Missing: []string{}}
		if len(docs) > 0 {
			out.Found = true
			leaves := map[string]ParameterValue{}
			flattenParams(docs[0], "", leaves)
			for _, p := range paths {
				hit := false
				for leaf, v := range leaves {
					if leaf == p || strings.HasPrefix(leaf, p+".") {
						out.Parameters[leaf] = v
						hit = true
					}
				}
				if !hit {
					out.Missing = append(out.Missing, p)
				}
			}
		} else {
			out.Missing = paths
		}

		b, _ := json.Marshal(out)
		text := "Cached parameter values: " + string(b)
		if !out.Found {
			text = fmt.Sprintf("No device with id %q. Copy ids from genieacs://devices/list or search_devices.", devID)
		}
		return mcp.NewToolResultStructured(out, text), nil
	}
	return tool, handler
}

// flattenParams collects the leaf parameters (nodes carrying _value) of a
// GenieACS device document. It accepts both the nested form GenieACS returns
// ({"Device":{"DeviceInfo":{...}}}) and dotted keys.
func flattenParams(node map[string]any, prefix string, out map[string]ParameterValue) {
	keys := make([]string, 0, len(node))
	for k := range node {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if strings.HasPrefix(k, "_") {
			continue
		}
		child, ok := node[k].(map[string]any)
		if !ok {
			continue
		}
		path := k
		if prefix != "" {
			path = prefix + "." + k
		}
		if v, ok := child["_value"]; ok {
			pv := ParameterValue{Value: v}
			pv.Type, _ = child["_type"].(string)
			pv.Timestamp, _ = child["_timestamp"].(string)
			out[path] = pv
		}
		flattenParams(child, path, out)
	}
}
