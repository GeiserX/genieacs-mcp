package tools

import (
	"context"
	"fmt"

	"github.com/geiserx/genieacs-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// NewRetryTask returns the retry_task tool, which clears the fault on a task and
// queues it again.
func NewRetryTask(acs *client.ACSClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := newTool("retry_task", hints("Retry faulted task", false, false, true),
		mcp.WithDescription(
			"Run a faulted task again: GenieACS clears the task's fault, and the device runs the task at its next session. "+
				"Read genieacs://faults/{device_id} first to see why it failed; retrying a task whose cause is unchanged faults again. "+
				"A task that has not faulted is left as it is. Example: retry_task(task_id=\"67abc123def4560012345678\").",
		),
		mcp.WithString("task_id",
			mcp.Required(),
			mcp.MinLength(1),
			mcp.Description("The faulted task's _id, from genieacs://tasks/{device_id}."),
		),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		taskID, err := req.RequireString("task_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		resp, err := acs.RetryTask(ctx, taskID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Task %s retried. ACS response: %s", taskID, string(resp))), nil
	}
	return tool, handler
}
