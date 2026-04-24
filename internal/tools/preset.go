package tools

import (
	"context"
	"fmt"

	"github.com/geiserx/genieacs-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func NewManagePreset(acs *client.ACSClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("manage_preset",
		mcp.WithDescription(
			"Create, update, or delete a GenieACS preset. Presets define automatic configuration "+
				"rules that are applied to CPE devices matching a precondition filter. "+
				"Use action=\"put\" to create or overwrite a preset, providing the full JSON body "+
				"with weight, precondition, and configurations. "+
				"Use action=\"delete\" to remove a preset by name. "+
				"A preset body should contain: weight (integer priority), precondition (a stringified "+
				"MongoDB-style JSON query, e.g. \"{\\\"_tags\\\":\\\"office\\\"}\"), and configurations "+
				"(array of objects with type \"value\", \"provision\", \"add_object\", or \"delete_object\"). "+
				"Example body: {\"weight\":0,\"precondition\":\"{\\\"_tags\\\":\\\"test\\\"}\",\"configurations\":[{\"type\":\"provision\",\"name\":\"myScript\"}]}. "+
				"Preset names cannot contain dots. "+
				"Use genieacs://presets/list to view existing presets before making changes. "+
				"Limitations: changes take effect on the next CPE inform, not immediately.",
		),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Description(
				"The operation to perform: \"put\" to create or update a preset, \"delete\" to remove it.",
			),
		),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description(
				"The preset name (used as the identifier in the URL). Cannot contain dots.",
			),
		),
		mcp.WithString("body",
			mcp.Description(
				"The full JSON preset document (required for action=\"put\", ignored for \"delete\"). "+
					"Must contain weight, precondition, and configurations fields.",
			),
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
			body, _ := req.GetArguments()["body"].(string)
			if body == "" {
				return mcp.NewToolResultError("body is required for action=\"put\""), nil
			}
			resp, err := acs.PutPreset(name, []byte(body))
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(
				fmt.Sprintf("Preset \"%s\" saved. Raw ACS response: %s", name, string(resp)),
			), nil
		case "delete":
			resp, err := acs.DeletePreset(name)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(
				fmt.Sprintf("Preset \"%s\" deleted. Raw ACS response: %s", name, string(resp)),
			), nil
		default:
			return mcp.NewToolResultError(fmt.Sprintf("unknown action %q — use \"put\" or \"delete\"", action)), nil
		}
	}

	return tool, handler
}
