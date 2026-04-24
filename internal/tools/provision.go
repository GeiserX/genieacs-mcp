package tools

import (
	"context"
	"fmt"

	"github.com/geiserx/genieacs-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func NewManageProvision(acs *client.ACSClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("manage_provision",
		mcp.WithDescription(
			"Create, update, or delete a GenieACS provision script. Provision scripts are JavaScript "+
				"functions that run on the ACS during a device's inform session to dynamically configure CPEs. "+
				"Use action=\"put\" to upload a new provision script or overwrite an existing one. "+
				"The script argument must be raw JavaScript source code (not JSON). "+
				"Use action=\"delete\" to remove a provision script by name. "+
				"GenieACS validates the script syntax on upload — a 400 error means the JavaScript has a syntax error. "+
				"Provision scripts are referenced by name in presets (via configurations of type \"provision\"). "+
				"Use genieacs://provisions/list to view existing scripts before making changes. "+
				"Example script: log(\"Device \" + args[0] + \" informed\"); "+
				"Limitations: provision scripts execute server-side in the GenieACS sandbox with a limited API "+
				"(declare, commit, ext, log). They cannot make arbitrary HTTP calls or access the filesystem.",
		),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Description(
				"The operation to perform: \"put\" to create or update a provision, \"delete\" to remove it.",
			),
		),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description(
				"The provision script name (used as the identifier in the URL).",
			),
		),
		mcp.WithString("script",
			mcp.Description(
				"The raw JavaScript source code of the provision script (required for action=\"put\", "+
					"ignored for \"delete\"). GenieACS validates syntax before saving.",
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
			script, _ := req.GetArguments()["script"].(string)
			if script == "" {
				return mcp.NewToolResultError("script is required for action=\"put\""), nil
			}
			resp, err := acs.PutProvision(name, script)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(
				fmt.Sprintf("Provision \"%s\" saved. Raw ACS response: %s", name, string(resp)),
			), nil
		case "delete":
			resp, err := acs.DeleteProvision(name)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(
				fmt.Sprintf("Provision \"%s\" deleted. Raw ACS response: %s", name, string(resp)),
			), nil
		default:
			return mcp.NewToolResultError(fmt.Sprintf("unknown action %q — use \"put\" or \"delete\"", action)), nil
		}
	}

	return tool, handler
}
