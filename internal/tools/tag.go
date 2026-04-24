package tools

import (
	"context"
	"fmt"

	"github.com/geiserx/genieacs-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func NewTagDevice(acs *client.ACSClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("tag_device",
		mcp.WithDescription(
			"Add or remove a tag on a CPE device in GenieACS. Tags are labels used to group "+
				"devices for preset matching, bulk operations, and organizational purposes. "+
				"Use action=\"add\" to tag a device and action=\"remove\" to untag it. "+
				"Tags are referenced in preset preconditions (e.g. {\"_tags\":\"office\"}) to target "+
				"specific device groups for automatic configuration. "+
				"The device must exist in GenieACS — returns a 404 error if the device ID is invalid. "+
				"Example: tag_device(device_id=\"00236A-SmartRG585-SMRT00236a42\", tag=\"office\", action=\"add\"). "+
				"Use search_devices with a _tags filter to verify tag assignment after modification. "+
				"Limitations: tag names are case-sensitive strings. There is no built-in tag listing — "+
				"use search_devices or genieacs://devices/list to discover existing tags on devices.",
		),
		mcp.WithString("device_id",
			mcp.Required(),
			mcp.Description(
				"The exact GenieACS device identifier (_id field). "+
					"Typically in the format OUI-ProductClass-SerialNumber (e.g. \"00236A-SmartRG585-SMRT00236a42\"). "+
					"Obtain valid IDs from the genieacs://devices/list resource.",
			),
		),
		mcp.WithString("tag",
			mcp.Required(),
			mcp.Description(
				"The tag string to add or remove (e.g. \"office\", \"floor-2\", \"firmware-pending\").",
			),
		),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Description(
				"The operation to perform: \"add\" to tag the device, \"remove\" to untag it.",
			),
		),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		devID, err := req.RequireString("device_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		tag, err := req.RequireString("tag")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		action, err := req.RequireString("action")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		switch action {
		case "add":
			resp, err := acs.AddTag(devID, tag)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(
				fmt.Sprintf("Tag \"%s\" added to device %s. Raw ACS response: %s", tag, devID, string(resp)),
			), nil
		case "remove":
			resp, err := acs.RemoveTag(devID, tag)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(
				fmt.Sprintf("Tag \"%s\" removed from device %s. Raw ACS response: %s", tag, devID, string(resp)),
			), nil
		default:
			return mcp.NewToolResultError(fmt.Sprintf("unknown action %q — use \"add\" or \"remove\"", action)), nil
		}
	}

	return tool, handler
}
