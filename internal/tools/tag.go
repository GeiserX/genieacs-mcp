package tools

import (
	"context"
	"fmt"

	"github.com/geiserx/genieacs-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewTagDevice returns the tag_device tool, which adds a tag to a device or
// removes one from it.
func NewTagDevice(acs *client.ACSClient) (mcp.Tool, server.ToolHandlerFunc) {
	// Removing a tag is not additive, and a preset matching on the tag may
	// reconfigure the device at its next inform, so the tool is destructive.
	tool := newTool("tag_device", hints("Add or remove device tag", false, true, true),
		mcp.WithDescription(
			"Add a tag to a CPE device or remove one. Tags group devices: presets match on them (precondition {\"_tags\":\"office\"}) "+
				"and search_devices finds by them. Adding or removing a tag can therefore make a preset apply to the device, "+
				"or stop applying, at its next inform. Tags are case-sensitive. Use letters, digits, \"-\" and \"_\": "+
				"GenieACS rejects a tag with other characters (a \".\" for example) with a 404. "+
				"Fails with ACS error 404 when the device id does not exist. "+
				"Example: tag_device(device_id=\"202BC1-BM632w-000001\", tag=\"office\", action=\"add\").",
		),
		deviceIDParam(),
		mcp.WithString("tag",
			mcp.Required(),
			mcp.MinLength(1),
			mcp.Pattern(`^[A-Za-z0-9_-]+$`),
			mcp.Description("The tag, e.g. \"office\", \"floor-2\", \"firmware_pending\"."),
		),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Enum("add", "remove"),
			mcp.Description("\"add\" tags the device, \"remove\" untags it."),
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
			resp, err := acs.AddTag(ctx, devID, tag)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(
				fmt.Sprintf("Tag %q added to device %s. ACS response: %s", tag, devID, string(resp)),
			), nil
		case "remove":
			resp, err := acs.RemoveTag(ctx, devID, tag)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(
				fmt.Sprintf("Tag %q removed from device %s. ACS response: %s", tag, devID, string(resp)),
			), nil
		default:
			return mcp.NewToolResultError(fmt.Sprintf("unknown action %q: use \"add\" or \"remove\"", action)), nil
		}
	}
	return tool, handler
}
