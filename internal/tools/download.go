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
		mcp.WithDescription(
			"Push a firmware image or configuration file to a CPE device via GenieACS TR-069 download mechanism. "+
				"Use this tool to initiate an OTA firmware upgrade or to push any file (config backups, supplementary data) to a device. "+
				"The file must already be uploaded to GenieACS — use genieacs://file/{name} to inspect available files. "+
				"Returns the raw JSON response from the ACS confirming the download task was queued. "+
				"Limitations: the download is asynchronous — the ACS queues the task and the CPE fetches the file "+
				"on its next session (periodic inform or connection request). Large firmware files may take minutes to transfer. "+
				"Verify success by checking genieacs://tasks/{device_id} after allowing time for the transfer. "+
				"Example: download_firmware(device_id=\"00236A-SmartRG585-SMRT00236a42\", file_id=\"firmware-v2.0.bin\").",
		),
		mcp.WithString("device_id",
			mcp.Required(),
			mcp.Description(
				"The exact GenieACS device identifier (_id field). "+
					"Typically in the format OUI-ProductClass-SerialNumber. "+
					"Obtain valid IDs from the genieacs://devices/list resource.",
			),
		),
		mcp.WithString("file_id",
			mcp.Required(),
			mcp.Description(
				"The GridFS _id (filename) of the file to push to the device. "+
					"Must match an existing file in GenieACS — use genieacs://file/{name} to verify it exists.",
			),
		),
		mcp.WithString("filename",
			mcp.Description(
				"Optional display name passed to the CPE during the download. "+
					"If omitted, the CPE receives the file_id as the filename.",
			),
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
