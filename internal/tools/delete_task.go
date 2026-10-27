package tools

import (
	"context"
	"fmt"

	"github.com/geiserx/genieacs-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewDeleteTask returns the delete_task tool, which removes a pending task from
// the GenieACS queue by its task ID.
func NewDeleteTask(acs *client.ACSClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := newTool("delete_task", hints("Delete queued task", false, true, true),
		mcp.WithDescription(
			"Remove a task from the GenieACS queue before the device runs it, for example a firmware download or reboot "+
				"queued by mistake, or a faulted task you do not want retried. GenieACS already deletes a task once the device "+
				"has run it, so every task you can see in genieacs://tasks/{device_id} is either waiting or faulted. "+
				"Fails with ACS error 503 while the device is in a session with GenieACS; try again a few seconds later. "+
				"Use retry_task instead to run a faulted task again. Example: delete_task(task_id=\"67abc123def4560012345678\").",
		),
		mcp.WithString("task_id",
			mcp.Required(),
			mcp.MinLength(1),
			mcp.Description("The task's _id, from genieacs://tasks/{device_id} or from the task_id a task tool returned."),
		),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		taskID, err := req.RequireString("task_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		resp, err := acs.DeleteTask(ctx, taskID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Task %s deleted. ACS response: %s", taskID, string(resp))), nil
	}
	return tool, handler
}
