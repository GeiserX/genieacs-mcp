package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/geiserx/genieacs-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func NewSearchDevices(acs *client.ACSClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("search_devices",
		mcp.WithDescription(
			"Search for CPE devices in GenieACS using MongoDB-style query filters. "+
				"Use this tool to find devices by tag, manufacturer, model, firmware version, "+
				"last inform time, or any other TR-069 parameter stored in the ACS. "+
				"The query argument is a JSON string using MongoDB query syntax. "+
				"Examples: {\"_tags\":\"office\"} to find devices tagged \"office\", "+
				"{\"InternetGatewayDevice.DeviceInfo.Manufacturer._value\":\"Huawei\"} to find Huawei devices, "+
				"{\"_lastInform\":{\"$lt\":\"2024-01-01T00:00:00Z\"}} to find devices that haven't informed since 2024. "+
				"For non-underscore-prefixed parameter paths, GenieACS automatically appends \"._value\" to the query, "+
				"so you can also query as {\"InternetGatewayDevice.DeviceInfo.Manufacturer\":\"Huawei\"}. "+
				"Returns a JSON array of matching device documents. "+
				"Use the limit argument to control the maximum number of results (default 50). "+
				"Limitations: complex aggregation queries are not supported — only standard MongoDB "+
				"comparison operators ($eq, $ne, $gt, $lt, $gte, $lte, $regex, $in, $nin, $exists).",
		),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description(
				"A MongoDB-style JSON query string to filter devices. "+
					"Example: {\"_tags\":\"office\"} or {\"_id\":\"00236A-SmartRG585-SMRT00236a42\"}",
			),
		),
		mcp.WithString("limit",
			mcp.Description(
				"Maximum number of devices to return. Defaults to 50 if not specified. "+
					"Use a higher value for bulk operations, but be mindful of response size.",
			),
		),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query, err := req.RequireString("query")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if !json.Valid([]byte(query)) {
			return mcp.NewToolResultError("query must be valid JSON"), nil
		}

		limit := 50
		if l, ok := req.GetArguments()["limit"].(string); ok && l != "" {
			if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
				limit = parsed
			}
		}

		resp, err := acs.SearchDevices(query, limit)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(
			fmt.Sprintf("Search results (limit %d): %s", limit, string(resp)),
		), nil
	}

	return tool, handler
}
