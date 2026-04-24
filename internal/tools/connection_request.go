package tools

import (
	"context"
	"fmt"

	"github.com/geiserx/genieacs-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func NewConnectionRequest(acs *client.ACSClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("connection_request",
		mcp.WithDescription(
			"Send a connection request to wake a CPE device without waiting for its periodic inform. "+
				"Use this tool when you need the device to contact the ACS immediately, for example "+
				"to execute pending tasks, apply preset changes, or force a parameter refresh. "+
				"This sends an HTTP connection request to the CPE's management URL as configured "+
				"in the TR-069 connection request mechanism. "+
				"Returns 200 on success (the CPE acknowledged the request) or 504 if the CPE is "+
				"unreachable (behind NAT, offline, or firewall blocking the connection request port). "+
				"Example: connection_request(device_id=\"00236A-SmartRG585-SMRT00236a42\"). "+
				"Limitations: the CPE must be network-reachable from the ACS for connection requests to work. "+
				"Devices behind NAT without STUN/NAT traversal configured will not respond. "+
				"This does not queue a task — it only triggers the CPE to initiate a CWMP session.",
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

		resp, err := acs.ConnectionRequest(devID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(
			fmt.Sprintf("Connection request sent to device %s. Raw ACS response: %s", devID, string(resp)),
		), nil
	}

	return tool, handler
}
