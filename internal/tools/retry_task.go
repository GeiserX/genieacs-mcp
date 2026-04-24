package tools

import (
	"context"
	"fmt"

	"github.com/geiserx/genieacs-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func NewRetryTask(acs *client.ACSClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("retry_task",
		mcp.WithDescription(
			"Retry a faulted task in GenieACS. Use this tool when a previously queued task "+
				"(reboot, firmware download, parameter set, etc.) has failed and you want to re-attempt it. "+
				"The task_id is the _id field from the task document, obtainable via the "+
				"genieacs://tasks/{id} resource — look for tasks with fault information. "+
				"This clears the fault and re-queues the task for execution on the next CPE inform. "+
				"Example: retry_task(task_id=\"67abc123def456\"). "+
				"Use genieacs://faults/{id} to understand why the task originally failed before retrying. "+
				"Limitations: only faulted tasks can be retried. Retrying a non-faulted task has no effect.",
		),
		mcp.WithString("task_id",
			mcp.Required(),
			mcp.Description(
				"The task identifier (_id field from the task document). "+
					"Obtain task IDs from the genieacs://tasks/{deviceId} resource.",
			),
		),
	)

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		taskID, err := req.RequireString("task_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		resp, err := acs.RetryTask(taskID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(
			fmt.Sprintf("Task %s retried. Raw ACS response: %s", taskID, string(resp)),
		), nil
	}

	return tool, handler
}
