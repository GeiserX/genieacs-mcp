// internal/tools/reboot.go
package tools // ← same package for every tool file

import (
	"context"
	"fmt"

	"github.com/geiserx/genieacs-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewReboot builds the Tool definition plus its handler.
// Call it from main.go and register both parts.
func NewReboot(acs *client.ACSClient) (mcp.Tool, server.ToolHandlerFunc) {

	tool := mcp.NewTool("reboot_device",
		mcp.WithDescription("Reboot a CPE via GenieACS"),
		mcp.WithString("device_id",
			mcp.Required(),
			mcp.Description("Exact device ID (_id) as known by GenieACS"),
		),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		devID, err := req.RequireString("device_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		resp, err := acs.RebootDevice(devID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(
			fmt.Sprintf("Reboot task queued/processed. Raw ACS response: %s", string(resp)),
		), nil
	}

	return tool, handler
}
