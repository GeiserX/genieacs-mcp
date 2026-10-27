package tools

import (
	"encoding/json"
	"fmt"

	"github.com/geiserx/genieacs-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Factory builds one tool definition and the handler that serves it.
type Factory func(*client.ACSClient) (mcp.Tool, server.ToolHandlerFunc)

// All returns every tool factory, in a fixed order. The server registers the
// ones its tool policy allows.
func All() []Factory {
	return []Factory{
		NewSearchDevices,
		NewGetParameter,
		NewRefreshParameter,
		NewSetParameter,
		NewReboot,
		NewDownloadFirmware,
		NewConnectionRequest,
		NewTagDevice,
		NewManagePreset,
		NewManageProvision,
		NewDeleteTask,
		NewRetryTask,
	}
}

// deviceIDDescription is shared by every tool that takes a device id.
const deviceIDDescription = "The GenieACS device id (the _id field), exactly as GenieACS stores it, " +
	"usually OUI-ProductClass-SerialNumber, e.g. \"202BC1-BM632w-000001\". GenieACS escapes characters " +
	"other than letters and digits inside the product class and serial as %XX (a \"-\" there becomes \"%2D\"), " +
	"so copy ids from genieacs://devices/list or search_devices rather than typing them."

func deviceIDParam() mcp.ToolOption {
	return mcp.WithString("device_id", mcp.Required(), mcp.MinLength(1), mcp.Description(deviceIDDescription))
}

// hints sets the title and all four annotation hints of a tool. Every tool
// talks to an external GenieACS, so openWorldHint is always true.
func hints(title string, readOnly, destructive, idempotent bool) []mcp.ToolOption {
	return []mcp.ToolOption{
		mcp.WithToolTitle(title),
		mcp.WithTitleAnnotation(title),
		mcp.WithReadOnlyHintAnnotation(readOnly),
		mcp.WithDestructiveHintAnnotation(destructive),
		mcp.WithIdempotentHintAnnotation(idempotent),
		mcp.WithOpenWorldHintAnnotation(true),
	}
}

// newTool is mcp.NewTool with the annotation options first.
func newTool(name string, annotations []mcp.ToolOption, opts ...mcp.ToolOption) mcp.Tool {
	return mcp.NewTool(name, append(annotations, opts...)...)
}

// TaskOutput is the structured result of a tool that queues a task on a device.
type TaskOutput struct {
	DeviceID string `json:"device_id" jsonschema_description:"The device the task was queued on."`
	TaskID   string `json:"task_id,omitempty" jsonschema_description:"The GenieACS task id; use it with delete_task or retry_task, or look for it in genieacs://tasks/{device_id}."`
	Status   string `json:"status" jsonschema:"enum=done,enum=queued" jsonschema_description:"done: the device ran the task in the session GenieACS opened at once. queued: the device did not answer within 3 seconds, so the task runs at its next inform or connection request."`
	Message  string `json:"message" jsonschema_description:"What happened, in one sentence."`
	Task     any    `json:"task,omitempty" jsonschema_description:"The task document GenieACS returned."`
}

// taskResult turns the NBI's answer to a queued task into a tool result.
func taskResult(deviceID, what string, r client.TaskResult) *mcp.CallToolResult {
	out := TaskOutput{DeviceID: deviceID, Status: "queued"}
	var doc map[string]any
	if json.Unmarshal(r.Body, &doc) == nil {
		out.Task = doc
		if id, ok := doc["_id"].(string); ok {
			out.TaskID = id
		}
	}
	if r.Done() {
		out.Status = "done"
		out.Message = fmt.Sprintf("%s ran on device %s.", what, deviceID)
	} else {
		out.Message = fmt.Sprintf("%s is queued on device %s: the device did not answer within 3 seconds, so it runs at the device's next inform or connection request. Check genieacs://tasks/%s and genieacs://faults/%s.", what, deviceID, deviceID, deviceID)
	}
	if out.TaskID != "" {
		out.Message += " Task id: " + out.TaskID + "."
	}
	return mcp.NewToolResultStructured(out, out.Message)
}

// stringOrJSON returns the argument as JSON text: a string is taken as JSON
// text already, any other value is encoded.
func stringOrJSON(v any) (string, bool) {
	switch x := v.(type) {
	case nil:
		return "", false
	case string:
		return x, x != ""
	default:
		b, err := json.Marshal(x)
		return string(b), err == nil
	}
}
