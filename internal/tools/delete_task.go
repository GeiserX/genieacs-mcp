package tools

import (
	"context"
	"fmt"

	"github.com/geiserx/genieacs-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func NewDeleteTask(acs *client.ACSClient) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("delete_task",
		mcp.WithDescription(
			"Delete a pending task from the GenieACS task queue. Use this tool to cancel a task "+
				"that was queued but has not yet been executed by the CPE, such as a mistakenly "+
				"queued firmware download or an unwanted reboot. "+
				"The task_id is the _id field from the task document, obtainable via the "+
				"genieacs://tasks/{id} resource. "+
				"Returns a 503 error if the device is currently in an active CWMP session "+
				"(the task cannot be deleted while the device is communicating with the ACS). "+
				"Example: delete_task(task_id=\"67abc123def456\"). "+
				"Limitations: only pending tasks can be deleted. Completed or in-progress tasks "+
				"cannot be removed. Use retry_task instead if the task faulted and you want to re-run it.",
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

		resp, err := acs.DeleteTask(taskID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(
			fmt.Sprintf("Task %s deleted. Raw ACS response: %s", taskID, string(resp)),
		), nil
	}

	return tool, handler
}
