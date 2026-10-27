package tools

import (
	"context"

	"github.com/geiserx/genieacs-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewDownloadFirmware returns the download_firmware tool.
func NewDownloadFirmware(acs *client.ACSClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := newTool("download_firmware", hints("Push firmware or file to device", false, true, false),
		mcp.WithDescription(
			"Tell a CPE device to download and apply a file stored on GenieACS: a firmware image, a configuration file "+
				"or another file the device accepts. A wrong image can leave a device unusable, so check the file's "+
				"metadata (genieacs://file/{name}: OUI, product class, version) against the device first. "+
				"GenieACS queues a download task and sends a connection request at once; the transfer itself can take "+
				"minutes after the tool returns. Check genieacs://tasks/{device_id} and genieacs://faults/{device_id} "+
				"afterwards. Example: download_firmware(device_id=\"202BC1-BM632w-000001\", file_id=\"firmware-v2.0.bin\").",
		),
		deviceIDParam(),
		mcp.WithString("file_id",
			mcp.Required(),
			mcp.MinLength(1),
			mcp.Description("The _id of the file on GenieACS, which is its filename, e.g. \"firmware-v2.0.bin\". "+
				"Check it exists with genieacs://file/{name}."),
		),
		mcp.WithString("filename",
			mcp.Description("Optional target file name sent to the device (TR-069 TargetFileName). "+
				"Omit it to let the device choose."),
		),
		mcp.WithOutputSchema[TaskOutput](),
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
		res, err := acs.DownloadFirmware(ctx, deviceID, fileID, req.GetString("filename", ""))
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return taskResult(deviceID, "Download of "+fileID, res), nil
	}
	return tool, handler
}
