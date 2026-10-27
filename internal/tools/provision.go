package tools

import (
	"context"
	"fmt"

	"github.com/geiserx/genieacs-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewManageProvision returns the manage_provision tool, which uploads, replaces
// or deletes a GenieACS provision script by name.
func NewManageProvision(acs *client.ACSClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := newTool("manage_provision", hints("Create, replace or delete provision script", false, true, true),
		mcp.WithDescription(
			"Create, replace or delete a GenieACS provision script. A provision is JavaScript that GenieACS runs during the "+
				"inform session of every device a preset applies it to, so replacing one changes how those devices are configured. "+
				"Scripts use the GenieACS sandbox API (declare, commit, clear, ext, log); ext() calls extension scripts that run on the "+
				"ACS host with full privileges, so treat a provision as code running on your ACS. action=\"put\" uploads or overwrites "+
				"the script (raw JavaScript, not JSON; GenieACS answers 400 on a syntax error); action=\"delete\" removes it. "+
				"Presets reference provisions by name. Read genieacs://provisions/list first. "+
				"Example script: log(\"Device \" + DeviceID.ID + \" informed\");",
		),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Enum("put", "delete"),
			mcp.Description("\"put\" creates or replaces the script, \"delete\" removes it."),
		),
		mcp.WithString("name",
			mcp.Required(),
			mcp.MinLength(1),
			mcp.Description("The provision name, its id on GenieACS."),
		),
		mcp.WithString("script",
			mcp.Description("For action=\"put\": the JavaScript source. Ignored for \"delete\"."),
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
			script := req.GetString("script", "")
			if script == "" {
				return mcp.NewToolResultError("script is required for action=\"put\""), nil
			}
			resp, err := acs.PutProvision(ctx, name, script)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("Provision %q saved. ACS response: %s", name, string(resp))), nil
		case "delete":
			resp, err := acs.DeleteProvision(ctx, name)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("Provision %q deleted. ACS response: %s", name, string(resp))), nil
		default:
			return mcp.NewToolResultError(fmt.Sprintf("unknown action %q: use \"put\" or \"delete\"", action)), nil
		}
	}
	return tool, handler
}
