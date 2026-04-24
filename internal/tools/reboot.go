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
		mcp.WithDescription(
			"Reboot a CPE device through the GenieACS TR-069 ACS. "+
				"Use this tool when a device needs to be restarted, for example after a configuration change, "+
				"firmware update, or to recover from an unresponsive state. "+
				"The device must exist in the GenieACS inventory — use genieacs://devices/list to discover valid IDs. "+
				"Returns the raw JSON response from the ACS confirming the task was queued. "+
				"Limitations: the reboot is asynchronous — the task is queued on the ACS and executed "+
				"the next time the CPE contacts the ACS (via its periodic inform or a connection request). "+
				"There is no confirmation that the device actually rebooted. "+
				"Example: reboot_device(device_id=\"00236A-SmartRG585-SMRT00236a42\").",
		),
		mcp.WithString("device_id",
			mcp.Required(),
			mcp.Description(
				"The exact GenieACS device identifier (_id field). "+
					"Typically in the format OUI-ProductClass-SerialNumber (e.g. \"00236A-SmartRG585-SMRT00236a42\"). "+
					"Obtain valid IDs from the genieacs://devices/list resource.",
			),
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
