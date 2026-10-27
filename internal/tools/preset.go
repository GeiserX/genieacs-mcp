package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/geiserx/genieacs-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewManagePreset returns the manage_preset tool, which creates, replaces or
// deletes a GenieACS preset by name.
func NewManagePreset(acs *client.ACSClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := newTool("manage_preset", hints("Create, replace or delete preset", false, true, true),
		mcp.WithDescription(
			"Create, replace or delete a GenieACS preset. A preset is a fleet-wide rule: at their next inform, every device "+
				"matching its precondition gets its configurations applied, so a broad precondition (or an empty one) "+
				"reconfigures the whole fleet. Confirm the precondition and the devices it matches (search_devices with the same "+
				"filter) before a put. action=\"put\" creates or overwrites the preset with body; action=\"delete\" removes it. "+
				"body is a JSON object: weight (integer, higher wins), precondition (a MongoDB-style filter as a JSON string, "+
				"e.g. \"{\\\"_tags\\\":\\\"office\\\"}\"), and configurations (an array of {type: \"value\" | \"provision\" | "+
				"\"add_object\" | \"delete_object\", ...}). Example body: "+
				"{\"weight\":0,\"precondition\":\"{\\\"_tags\\\":\\\"pilot\\\"}\",\"configurations\":[{\"type\":\"provision\",\"name\":\"myScript\"}]}. "+
				"Read genieacs://presets/list first.",
		),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Enum("put", "delete"),
			mcp.Description("\"put\" creates or replaces the preset, \"delete\" removes it."),
		),
		mcp.WithString("name",
			mcp.Required(),
			mcp.MinLength(1),
			mcp.Pattern(`^[^.]+$`),
			mcp.Description("The preset name, its id on GenieACS. It cannot contain a dot."),
		),
		mcp.WithString("body",
			mcp.Description("For action=\"put\": the preset document as a JSON object string, with weight, precondition and configurations. Ignored for \"delete\"."),
		),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		action, err := req.RequireString("action")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		name, err := req.RequireString("name")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		switch action {
		case "put":
			body, ok := stringOrJSON(req.GetArguments()["body"])
			if !ok {
				return mcp.NewToolResultError("body is required for action=\"put\""), nil
			}
			var doc map[string]any
			if err := json.Unmarshal([]byte(body), &doc); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("body must be a JSON object (the preset document): %v", err)), nil
			}
			resp, err := acs.PutPreset(ctx, name, []byte(body))
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("Preset %q saved. ACS response: %s", name, string(resp))), nil
		case "delete":
			resp, err := acs.DeletePreset(ctx, name)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("Preset %q deleted. ACS response: %s", name, string(resp))), nil
		default:
			return mcp.NewToolResultError(fmt.Sprintf("unknown action %q: use \"put\" or \"delete\"", action)), nil
		}
	}
	return tool, handler
}
