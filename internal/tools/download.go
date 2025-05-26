// internal/tools/download.go
package tools

import (
	"context"
	"fmt"

	"github.com/geiserx/genieacs-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewDownloadFirmware defines the tool and its handler.
func NewDownloadFirmware(acs *client.ACSClient) (mcp.Tool, server.ToolHandlerFunc) {

	tool := mcp.NewTool("download_firmware",
		mcp.WithDescription("Push a firmware (or any file) to a device"),
		mcp.WithString("device_id",
			mcp.Required(),
			mcp.Description("Target device ID"),
		),
		mcp.WithString("file_id",
			mcp.Required(),
			mcp.Description("GridFS _id of the file to download"),
		),
		mcp.WithString("filename",
			mcp.Description("(optional) file name to pass to the CPE"),
		),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		deviceID, err := req.RequireString("device_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		fileID, err := req.RequireString("file_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		name, _ := req.GetArguments()["filename"].(string)

		resp, err := acs.DownloadFirmware(deviceID, fileID, name)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(
			fmt.Sprintf("Download task queued/processed. Raw ACS response: %s", string(resp)),
		), nil
	}

	return tool, handler
}
