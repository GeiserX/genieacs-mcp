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
		mcp.WithDescription("Ask the CPE to send an updated value for a single TR-069 parameter"),
		mcp.WithString("device_id",
			mcp.Required(),
			mcp.Description("Target device ID"),
		),
		mcp.WithString("parameter",
			mcp.Required(),
			mcp.Description("Full TR-069 path, e.g. Device.DeviceInfo.SerialNumber"),
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
