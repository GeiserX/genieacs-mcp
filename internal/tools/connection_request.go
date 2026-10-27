package tools

import (
	"context"
	"fmt"

	"github.com/geiserx/genieacs-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewConnectionRequest returns the connection_request tool, which asks GenieACS to
// wake a CPE now without queuing a task.
func NewConnectionRequest(acs *client.ACSClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := newTool("connection_request", hints("Wake device (connection request)", false, false, true),
		mcp.WithDescription(
			"Send a TR-069 connection request so a CPE device contacts GenieACS now instead of at its next periodic inform. "+
				"Use it to run tasks that are already queued, or to apply a preset change without waiting. It queues nothing itself. "+
				"Succeeds when the device answered; fails with ACS error 504 when the device is unreachable "+
				"(offline, or behind NAT without STUN). Example: connection_request(device_id=\"202BC1-BM632w-000001\").",
		),
		deviceIDParam(),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		devID, err := req.RequireString("device_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		resp, err := acs.ConnectionRequest(ctx, devID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(
			fmt.Sprintf("Connection request sent to device %s; the device answered. ACS response: %s", devID, string(resp)),
		), nil
	}
	return tool, handler
}
