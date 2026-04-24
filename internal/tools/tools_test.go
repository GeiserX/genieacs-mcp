package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/geiserx/genieacs-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// ---------- helpers ----------

// newTestACS spins up a local HTTP server and returns the ACSClient + cleanup func.
func newTestACS(t *testing.T, handler http.HandlerFunc) (*client.ACSClient, func()) {
	t.Helper()
	srv := httptest.NewServer(handler)
	acs := client.NewACS(srv.URL, "", "")
	return acs, srv.Close
}

// newTestACSWithAuth is like newTestACS but uses basic-auth.
func newTestACSWithAuth(t *testing.T, handler http.HandlerFunc) (*client.ACSClient, func()) {
	t.Helper()
	srv := httptest.NewServer(handler)
	acs := client.NewACS(srv.URL, "admin", "secret")
	return acs, srv.Close
}

// callTool is a shorthand that builds a CallToolRequest and invokes a handler.
func callTool(t *testing.T, handler func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error), args map[string]any) *mcp.CallToolResult {
	t.Helper()
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler returned unexpected protocol error: %v", err)
	}
	return result
}

// resultText extracts the text from the first TextContent in a CallToolResult.
func resultText(t *testing.T, r *mcp.CallToolResult) string {
	t.Helper()
	if len(r.Content) == 0 {
		t.Fatal("expected at least one content item in result")
	}
	tc, ok := r.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", r.Content[0])
	}
	return tc.Text
}

// okHandler returns 200 with the given body for any request.
func okHandler(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(body))
	}
}

// errHandler returns the given status code + body.
func errHandler(code int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(code)
		w.Write([]byte(body))
	}
}

// ---------- NewReboot ----------

func TestReboot_returns_success_when_ACS_accepts_task(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`{"_id":"task-1"}`))
	defer cleanup()

	_, handler := NewReboot(acs)
	result := callTool(t, handler, map[string]any{"device_id": "dev-001"})

	if result.IsError {
		t.Fatal("expected success, got error")
	}
	text := resultText(t, result)
	if !strings.Contains(text, "task-1") {
		t.Errorf("expected response to contain task-1, got: %s", text)
	}
}

func TestReboot_returns_error_when_device_id_missing(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`{}`))
	defer cleanup()

	_, handler := NewReboot(acs)
	result := callTool(t, handler, map[string]any{})

	if !result.IsError {
		t.Fatal("expected error result when device_id is missing")
	}
}

func TestReboot_returns_error_when_ACS_fails(t *testing.T) {
	acs, cleanup := newTestACS(t, errHandler(500, "boom"))
	defer cleanup()

	_, handler := NewReboot(acs)
	result := callTool(t, handler, map[string]any{"device_id": "dev-001"})

	if !result.IsError {
		t.Fatal("expected error result on ACS failure")
	}
	text := resultText(t, result)
	if !strings.Contains(text, "500") {
		t.Errorf("expected error to mention 500, got: %s", text)
	}
}

// ---------- NewDownloadFirmware ----------

func TestDownloadFirmware_returns_success_with_all_params(t *testing.T) {
	acs, cleanup := newTestACS(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["name"] != "download" {
			t.Errorf("expected download task, got %v", body["name"])
		}
		if body["file"] != "fw-001" {
			t.Errorf("expected file fw-001, got %v", body["file"])
		}
		if body["targetFileName"] != "firmware.bin" {
			t.Errorf("expected targetFileName firmware.bin, got %v", body["targetFileName"])
		}
		w.Write([]byte(`{"_id":"task-2"}`))
	})
	defer cleanup()

	_, handler := NewDownloadFirmware(acs)
	result := callTool(t, handler, map[string]any{
		"device_id": "dev-001",
		"file_id":   "fw-001",
		"filename":  "firmware.bin",
	})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}
	if !strings.Contains(resultText(t, result), "task-2") {
		t.Error("expected response to contain task ID")
	}
}

func TestDownloadFirmware_works_without_optional_filename(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`{"_id":"task-3"}`))
	defer cleanup()

	_, handler := NewDownloadFirmware(acs)
	result := callTool(t, handler, map[string]any{
		"device_id": "dev-001",
		"file_id":   "fw-001",
	})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}
}

func TestDownloadFirmware_returns_error_when_device_id_missing(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`{}`))
	defer cleanup()

	_, handler := NewDownloadFirmware(acs)
	result := callTool(t, handler, map[string]any{"file_id": "fw-001"})

	if !result.IsError {
		t.Fatal("expected error when device_id missing")
	}
}

func TestDownloadFirmware_returns_error_when_file_id_missing(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`{}`))
	defer cleanup()

	_, handler := NewDownloadFirmware(acs)
	result := callTool(t, handler, map[string]any{"device_id": "dev-001"})

	if !result.IsError {
		t.Fatal("expected error when file_id missing")
	}
}

func TestDownloadFirmware_returns_error_when_ACS_fails(t *testing.T) {
	acs, cleanup := newTestACS(t, errHandler(500, "storage error"))
	defer cleanup()

	_, handler := NewDownloadFirmware(acs)
	result := callTool(t, handler, map[string]any{
		"device_id": "dev-001",
		"file_id":   "fw-001",
	})

	if !result.IsError {
		t.Fatal("expected error on ACS failure")
	}
}

// ---------- NewRefreshParameter ----------

func TestRefreshParameter_returns_success(t *testing.T) {
	acs, cleanup := newTestACS(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["name"] != "getParameterValues" {
			t.Errorf("expected getParameterValues, got %v", body["name"])
		}
		w.Write([]byte(`{"_id":"task-4"}`))
	})
	defer cleanup()

	_, handler := NewRefreshParameter(acs)
	result := callTool(t, handler, map[string]any{
		"device_id": "dev-001",
		"parameter": "Device.DeviceInfo.SoftwareVersion",
	})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}
	if !strings.Contains(resultText(t, result), "task-4") {
		t.Error("expected response to contain task ID")
	}
}

func TestRefreshParameter_returns_error_when_device_id_missing(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`{}`))
	defer cleanup()

	_, handler := NewRefreshParameter(acs)
	result := callTool(t, handler, map[string]any{"parameter": "Device.Foo"})

	if !result.IsError {
		t.Fatal("expected error when device_id missing")
	}
}

func TestRefreshParameter_returns_error_when_parameter_missing(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`{}`))
	defer cleanup()

	_, handler := NewRefreshParameter(acs)
	result := callTool(t, handler, map[string]any{"device_id": "dev-001"})

	if !result.IsError {
		t.Fatal("expected error when parameter missing")
	}
}

func TestRefreshParameter_returns_error_when_ACS_fails(t *testing.T) {
	acs, cleanup := newTestACS(t, errHandler(504, "timeout"))
	defer cleanup()

	_, handler := NewRefreshParameter(acs)
	result := callTool(t, handler, map[string]any{
		"device_id": "dev-001",
		"parameter": "Device.Foo",
	})

	if !result.IsError {
		t.Fatal("expected error on ACS failure")
	}
}

// ---------- NewSetParameter ----------

func TestSetParameter_returns_success_with_valid_JSON(t *testing.T) {
	acs, cleanup := newTestACS(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["name"] != "setParameterValues" {
			t.Errorf("expected setParameterValues, got %v", body["name"])
		}
		w.Write([]byte(`{"_id":"task-5"}`))
	})
	defer cleanup()

	_, handler := NewSetParameter(acs)
	result := callTool(t, handler, map[string]any{
		"device_id":        "dev-001",
		"parameter_values": `[["Device.WiFi.SSID","TestSSID","xsd:string"]]`,
	})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}
	if !strings.Contains(resultText(t, result), "task-5") {
		t.Error("expected response to contain task ID")
	}
}

func TestSetParameter_returns_error_when_device_id_missing(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`{}`))
	defer cleanup()

	_, handler := NewSetParameter(acs)
	result := callTool(t, handler, map[string]any{
		"parameter_values": `[["foo","bar"]]`,
	})

	if !result.IsError {
		t.Fatal("expected error when device_id missing")
	}
}

func TestSetParameter_returns_error_when_parameter_values_missing(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`{}`))
	defer cleanup()

	_, handler := NewSetParameter(acs)
	result := callTool(t, handler, map[string]any{"device_id": "dev-001"})

	if !result.IsError {
		t.Fatal("expected error when parameter_values missing")
	}
}

func TestSetParameter_returns_error_when_parameter_values_invalid_JSON(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`{}`))
	defer cleanup()

	_, handler := NewSetParameter(acs)
	result := callTool(t, handler, map[string]any{
		"device_id":        "dev-001",
		"parameter_values": "not-valid-json{{{",
	})

	if !result.IsError {
		t.Fatal("expected error for invalid JSON")
	}
	text := resultText(t, result)
	if !strings.Contains(text, "not valid JSON") {
		t.Errorf("expected JSON validation error, got: %s", text)
	}
}

func TestSetParameter_returns_error_when_ACS_fails(t *testing.T) {
	acs, cleanup := newTestACS(t, errHandler(500, "db error"))
	defer cleanup()

	_, handler := NewSetParameter(acs)
	result := callTool(t, handler, map[string]any{
		"device_id":        "dev-001",
		"parameter_values": `[["foo","bar"]]`,
	})

	if !result.IsError {
		t.Fatal("expected error on ACS failure")
	}
}

// ---------- NewGetParameter ----------

func TestGetParameter_returns_cached_values(t *testing.T) {
	acs, cleanup := newTestACS(t, func(w http.ResponseWriter, r *http.Request) {
		proj := r.URL.Query().Get("projection")
		if proj != "Device.DeviceInfo.SoftwareVersion" {
			t.Errorf("unexpected projection: %s", proj)
		}
		w.Write([]byte(`[{"_id":"dev-001","Device.DeviceInfo.SoftwareVersion":{"_value":"1.0"}}]`))
	})
	defer cleanup()

	_, handler := NewGetParameter(acs)
	result := callTool(t, handler, map[string]any{
		"device_id":      "dev-001",
		"parameter_path": "Device.DeviceInfo.SoftwareVersion",
	})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}
	text := resultText(t, result)
	if !strings.Contains(text, "1.0") {
		t.Errorf("expected response to contain value 1.0, got: %s", text)
	}
}

func TestGetParameter_returns_error_when_device_id_missing(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`{}`))
	defer cleanup()

	_, handler := NewGetParameter(acs)
	result := callTool(t, handler, map[string]any{"parameter_path": "Device.Foo"})

	if !result.IsError {
		t.Fatal("expected error when device_id missing")
	}
}

func TestGetParameter_returns_error_when_parameter_path_missing(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`{}`))
	defer cleanup()

	_, handler := NewGetParameter(acs)
	result := callTool(t, handler, map[string]any{"device_id": "dev-001"})

	if !result.IsError {
		t.Fatal("expected error when parameter_path missing")
	}
}

func TestGetParameter_returns_error_when_ACS_fails(t *testing.T) {
	acs, cleanup := newTestACS(t, errHandler(404, "not found"))
	defer cleanup()

	_, handler := NewGetParameter(acs)
	result := callTool(t, handler, map[string]any{
		"device_id":      "dev-001",
		"parameter_path": "Device.Foo",
	})

	if !result.IsError {
		t.Fatal("expected error on ACS failure")
	}
}

// ---------- NewManagePreset ----------

func TestManagePreset_put_creates_preset(t *testing.T) {
	acs, cleanup := newTestACS(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/presets/") {
			t.Errorf("expected /presets/ path, got %s", r.URL.Path)
		}
		w.Write([]byte(`{}`))
	})
	defer cleanup()

	_, handler := NewManagePreset(acs)
	result := callTool(t, handler, map[string]any{
		"action": "put",
		"name":   "test-preset",
		"body":   `{"weight":0,"precondition":"{}","configurations":[]}`,
	})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}
	text := resultText(t, result)
	if !strings.Contains(text, "test-preset") {
		t.Errorf("expected response to mention preset name, got: %s", text)
	}
	if !strings.Contains(text, "saved") {
		t.Errorf("expected response to confirm save, got: %s", text)
	}
}

func TestManagePreset_put_returns_error_when_body_empty(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`{}`))
	defer cleanup()

	_, handler := NewManagePreset(acs)
	result := callTool(t, handler, map[string]any{
		"action": "put",
		"name":   "test-preset",
	})

	if !result.IsError {
		t.Fatal("expected error when body is missing for put action")
	}
	text := resultText(t, result)
	if !strings.Contains(text, "body is required") {
		t.Errorf("expected 'body is required' error, got: %s", text)
	}
}

func TestManagePreset_delete_removes_preset(t *testing.T) {
	acs, cleanup := newTestACS(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(200)
	})
	defer cleanup()

	_, handler := NewManagePreset(acs)
	result := callTool(t, handler, map[string]any{
		"action": "delete",
		"name":   "test-preset",
	})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}
	text := resultText(t, result)
	if !strings.Contains(text, "deleted") {
		t.Errorf("expected response to confirm deletion, got: %s", text)
	}
}

func TestManagePreset_returns_error_for_unknown_action(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`{}`))
	defer cleanup()

	_, handler := NewManagePreset(acs)
	result := callTool(t, handler, map[string]any{
		"action": "patch",
		"name":   "test-preset",
	})

	if !result.IsError {
		t.Fatal("expected error for unknown action")
	}
	text := resultText(t, result)
	if !strings.Contains(text, "unknown action") {
		t.Errorf("expected 'unknown action' error, got: %s", text)
	}
}

func TestManagePreset_returns_error_when_action_missing(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`{}`))
	defer cleanup()

	_, handler := NewManagePreset(acs)
	result := callTool(t, handler, map[string]any{"name": "test-preset"})

	if !result.IsError {
		t.Fatal("expected error when action missing")
	}
}

func TestManagePreset_returns_error_when_name_missing(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`{}`))
	defer cleanup()

	_, handler := NewManagePreset(acs)
	result := callTool(t, handler, map[string]any{"action": "put"})

	if !result.IsError {
		t.Fatal("expected error when name missing")
	}
}

func TestManagePreset_put_returns_error_when_ACS_fails(t *testing.T) {
	acs, cleanup := newTestACS(t, errHandler(500, "db error"))
	defer cleanup()

	_, handler := NewManagePreset(acs)
	result := callTool(t, handler, map[string]any{
		"action": "put",
		"name":   "test-preset",
		"body":   `{"weight":0}`,
	})

	if !result.IsError {
		t.Fatal("expected error on ACS failure")
	}
}

func TestManagePreset_delete_returns_error_when_ACS_fails(t *testing.T) {
	acs, cleanup := newTestACS(t, errHandler(404, "not found"))
	defer cleanup()

	_, handler := NewManagePreset(acs)
	result := callTool(t, handler, map[string]any{
		"action": "delete",
		"name":   "nonexistent",
	})

	if !result.IsError {
		t.Fatal("expected error on ACS failure")
	}
}

// ---------- NewManageProvision ----------

func TestManageProvision_put_creates_provision(t *testing.T) {
	acs, cleanup := newTestACS(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/javascript" {
			t.Errorf("expected application/javascript, got %s", r.Header.Get("Content-Type"))
		}
		w.Write([]byte(`{}`))
	})
	defer cleanup()

	_, handler := NewManageProvision(acs)
	result := callTool(t, handler, map[string]any{
		"action": "put",
		"name":   "my-prov",
		"script": "log('hello');",
	})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}
	text := resultText(t, result)
	if !strings.Contains(text, "my-prov") {
		t.Errorf("expected response to mention provision name, got: %s", text)
	}
	if !strings.Contains(text, "saved") {
		t.Errorf("expected response to confirm save, got: %s", text)
	}
}

func TestManageProvision_put_returns_error_when_script_empty(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`{}`))
	defer cleanup()

	_, handler := NewManageProvision(acs)
	result := callTool(t, handler, map[string]any{
		"action": "put",
		"name":   "my-prov",
	})

	if !result.IsError {
		t.Fatal("expected error when script is missing for put action")
	}
	text := resultText(t, result)
	if !strings.Contains(text, "script is required") {
		t.Errorf("expected 'script is required' error, got: %s", text)
	}
}

func TestManageProvision_delete_removes_provision(t *testing.T) {
	acs, cleanup := newTestACS(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(200)
	})
	defer cleanup()

	_, handler := NewManageProvision(acs)
	result := callTool(t, handler, map[string]any{
		"action": "delete",
		"name":   "my-prov",
	})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}
	text := resultText(t, result)
	if !strings.Contains(text, "deleted") {
		t.Errorf("expected response to confirm deletion, got: %s", text)
	}
}

func TestManageProvision_returns_error_for_unknown_action(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`{}`))
	defer cleanup()

	_, handler := NewManageProvision(acs)
	result := callTool(t, handler, map[string]any{
		"action": "update",
		"name":   "my-prov",
	})

	if !result.IsError {
		t.Fatal("expected error for unknown action")
	}
	text := resultText(t, result)
	if !strings.Contains(text, "unknown action") {
		t.Errorf("expected 'unknown action' error, got: %s", text)
	}
}

func TestManageProvision_returns_error_when_action_missing(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`{}`))
	defer cleanup()

	_, handler := NewManageProvision(acs)
	result := callTool(t, handler, map[string]any{"name": "my-prov"})

	if !result.IsError {
		t.Fatal("expected error when action missing")
	}
}

func TestManageProvision_returns_error_when_name_missing(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`{}`))
	defer cleanup()

	_, handler := NewManageProvision(acs)
	result := callTool(t, handler, map[string]any{"action": "delete"})

	if !result.IsError {
		t.Fatal("expected error when name missing")
	}
}

func TestManageProvision_put_returns_error_when_ACS_fails(t *testing.T) {
	acs, cleanup := newTestACS(t, errHandler(400, "syntax error"))
	defer cleanup()

	_, handler := NewManageProvision(acs)
	result := callTool(t, handler, map[string]any{
		"action": "put",
		"name":   "my-prov",
		"script": "invalid{{{",
	})

	if !result.IsError {
		t.Fatal("expected error on ACS failure")
	}
}

func TestManageProvision_delete_returns_error_when_ACS_fails(t *testing.T) {
	acs, cleanup := newTestACS(t, errHandler(404, "not found"))
	defer cleanup()

	_, handler := NewManageProvision(acs)
	result := callTool(t, handler, map[string]any{
		"action": "delete",
		"name":   "nonexistent",
	})

	if !result.IsError {
		t.Fatal("expected error on ACS failure")
	}
}

// ---------- NewSearchDevices ----------

func TestSearchDevices_returns_matching_devices(t *testing.T) {
	acs, cleanup := newTestACS(t, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		if query != `{"_tags":"office"}` {
			t.Errorf("unexpected query: %s", query)
		}
		limit := r.URL.Query().Get("limit")
		if limit != "50" {
			t.Errorf("expected default limit 50, got %s", limit)
		}
		w.Write([]byte(`[{"_id":"dev-001"}]`))
	})
	defer cleanup()

	_, handler := NewSearchDevices(acs)
	result := callTool(t, handler, map[string]any{
		"query": `{"_tags":"office"}`,
	})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}
	text := resultText(t, result)
	if !strings.Contains(text, "dev-001") {
		t.Errorf("expected response to contain device ID, got: %s", text)
	}
}

func TestSearchDevices_respects_custom_limit(t *testing.T) {
	acs, cleanup := newTestACS(t, func(w http.ResponseWriter, r *http.Request) {
		limit := r.URL.Query().Get("limit")
		if limit != "10" {
			t.Errorf("expected limit 10, got %s", limit)
		}
		w.Write([]byte(`[]`))
	})
	defer cleanup()

	_, handler := NewSearchDevices(acs)
	result := callTool(t, handler, map[string]any{
		"query": `{"_tags":"test"}`,
		"limit": "10",
	})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}
	text := resultText(t, result)
	if !strings.Contains(text, "limit 10") {
		t.Errorf("expected response to show limit 10, got: %s", text)
	}
}

func TestSearchDevices_uses_default_limit_for_invalid_value(t *testing.T) {
	acs, cleanup := newTestACS(t, func(w http.ResponseWriter, r *http.Request) {
		limit := r.URL.Query().Get("limit")
		if limit != "50" {
			t.Errorf("expected default limit 50 for invalid input, got %s", limit)
		}
		w.Write([]byte(`[]`))
	})
	defer cleanup()

	_, handler := NewSearchDevices(acs)
	result := callTool(t, handler, map[string]any{
		"query": `{"_id":"x"}`,
		"limit": "notanumber",
	})

	if result.IsError {
		t.Fatalf("expected success even with invalid limit, got error: %s", resultText(t, result))
	}
}

func TestSearchDevices_returns_error_when_query_missing(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`[]`))
	defer cleanup()

	_, handler := NewSearchDevices(acs)
	result := callTool(t, handler, map[string]any{})

	if !result.IsError {
		t.Fatal("expected error when query missing")
	}
}

func TestSearchDevices_returns_error_when_query_invalid_JSON(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`[]`))
	defer cleanup()

	_, handler := NewSearchDevices(acs)
	result := callTool(t, handler, map[string]any{
		"query": "not-json{{{",
	})

	if !result.IsError {
		t.Fatal("expected error for invalid JSON query")
	}
	text := resultText(t, result)
	if !strings.Contains(text, "valid JSON") {
		t.Errorf("expected JSON validation error, got: %s", text)
	}
}

func TestSearchDevices_returns_error_when_ACS_fails(t *testing.T) {
	acs, cleanup := newTestACS(t, errHandler(500, "timeout"))
	defer cleanup()

	_, handler := NewSearchDevices(acs)
	result := callTool(t, handler, map[string]any{
		"query": `{"_id":"dev-001"}`,
	})

	if !result.IsError {
		t.Fatal("expected error on ACS failure")
	}
}

// ---------- NewTagDevice ----------

func TestTagDevice_add_succeeds(t *testing.T) {
	acs, cleanup := newTestACS(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST for add, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/tags/") {
			t.Errorf("expected /tags/ in path, got %s", r.URL.Path)
		}
		w.WriteHeader(200)
	})
	defer cleanup()

	_, handler := NewTagDevice(acs)
	result := callTool(t, handler, map[string]any{
		"device_id": "dev-001",
		"tag":       "office",
		"action":    "add",
	})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}
	text := resultText(t, result)
	if !strings.Contains(text, "added") {
		t.Errorf("expected 'added' in response, got: %s", text)
	}
}

func TestTagDevice_remove_succeeds(t *testing.T) {
	acs, cleanup := newTestACS(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE for remove, got %s", r.Method)
		}
		w.WriteHeader(200)
	})
	defer cleanup()

	_, handler := NewTagDevice(acs)
	result := callTool(t, handler, map[string]any{
		"device_id": "dev-001",
		"tag":       "office",
		"action":    "remove",
	})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}
	text := resultText(t, result)
	if !strings.Contains(text, "removed") {
		t.Errorf("expected 'removed' in response, got: %s", text)
	}
}

func TestTagDevice_returns_error_for_unknown_action(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`{}`))
	defer cleanup()

	_, handler := NewTagDevice(acs)
	result := callTool(t, handler, map[string]any{
		"device_id": "dev-001",
		"tag":       "office",
		"action":    "toggle",
	})

	if !result.IsError {
		t.Fatal("expected error for unknown action")
	}
	text := resultText(t, result)
	if !strings.Contains(text, "unknown action") {
		t.Errorf("expected 'unknown action' error, got: %s", text)
	}
}

func TestTagDevice_returns_error_when_device_id_missing(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`{}`))
	defer cleanup()

	_, handler := NewTagDevice(acs)
	result := callTool(t, handler, map[string]any{
		"tag":    "office",
		"action": "add",
	})

	if !result.IsError {
		t.Fatal("expected error when device_id missing")
	}
}

func TestTagDevice_returns_error_when_tag_missing(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`{}`))
	defer cleanup()

	_, handler := NewTagDevice(acs)
	result := callTool(t, handler, map[string]any{
		"device_id": "dev-001",
		"action":    "add",
	})

	if !result.IsError {
		t.Fatal("expected error when tag missing")
	}
}

func TestTagDevice_returns_error_when_action_missing(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`{}`))
	defer cleanup()

	_, handler := NewTagDevice(acs)
	result := callTool(t, handler, map[string]any{
		"device_id": "dev-001",
		"tag":       "office",
	})

	if !result.IsError {
		t.Fatal("expected error when action missing")
	}
}

func TestTagDevice_add_returns_error_when_ACS_fails(t *testing.T) {
	acs, cleanup := newTestACS(t, errHandler(404, "device not found"))
	defer cleanup()

	_, handler := NewTagDevice(acs)
	result := callTool(t, handler, map[string]any{
		"device_id": "nonexistent",
		"tag":       "office",
		"action":    "add",
	})

	if !result.IsError {
		t.Fatal("expected error on ACS failure")
	}
}

func TestTagDevice_remove_returns_error_when_ACS_fails(t *testing.T) {
	acs, cleanup := newTestACS(t, errHandler(404, "device not found"))
	defer cleanup()

	_, handler := NewTagDevice(acs)
	result := callTool(t, handler, map[string]any{
		"device_id": "nonexistent",
		"tag":       "office",
		"action":    "remove",
	})

	if !result.IsError {
		t.Fatal("expected error on ACS failure")
	}
}

// ---------- NewConnectionRequest ----------

func TestConnectionRequest_returns_success(t *testing.T) {
	acs, cleanup := newTestACS(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !r.URL.Query().Has("connection_request") {
			t.Error("expected connection_request query param")
		}
		w.WriteHeader(200)
	})
	defer cleanup()

	_, handler := NewConnectionRequest(acs)
	result := callTool(t, handler, map[string]any{"device_id": "dev-001"})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}
	text := resultText(t, result)
	if !strings.Contains(text, "dev-001") {
		t.Errorf("expected device ID in response, got: %s", text)
	}
}

func TestConnectionRequest_returns_error_when_device_id_missing(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`{}`))
	defer cleanup()

	_, handler := NewConnectionRequest(acs)
	result := callTool(t, handler, map[string]any{})

	if !result.IsError {
		t.Fatal("expected error when device_id missing")
	}
}

func TestConnectionRequest_returns_error_when_ACS_fails(t *testing.T) {
	acs, cleanup := newTestACS(t, errHandler(504, "CPE unreachable"))
	defer cleanup()

	_, handler := NewConnectionRequest(acs)
	result := callTool(t, handler, map[string]any{"device_id": "dev-001"})

	if !result.IsError {
		t.Fatal("expected error on ACS failure")
	}
}

// ---------- NewDeleteTask ----------

func TestDeleteTask_returns_success(t *testing.T) {
	acs, cleanup := newTestACS(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/tasks/") {
			t.Errorf("expected /tasks/ path, got %s", r.URL.Path)
		}
		w.WriteHeader(200)
	})
	defer cleanup()

	_, handler := NewDeleteTask(acs)
	result := callTool(t, handler, map[string]any{"task_id": "task-001"})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}
	text := resultText(t, result)
	if !strings.Contains(text, "task-001") {
		t.Errorf("expected task ID in response, got: %s", text)
	}
	if !strings.Contains(text, "deleted") {
		t.Errorf("expected 'deleted' in response, got: %s", text)
	}
}

func TestDeleteTask_returns_error_when_task_id_missing(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`{}`))
	defer cleanup()

	_, handler := NewDeleteTask(acs)
	result := callTool(t, handler, map[string]any{})

	if !result.IsError {
		t.Fatal("expected error when task_id missing")
	}
}

func TestDeleteTask_returns_error_when_ACS_fails(t *testing.T) {
	acs, cleanup := newTestACS(t, errHandler(503, "device in session"))
	defer cleanup()

	_, handler := NewDeleteTask(acs)
	result := callTool(t, handler, map[string]any{"task_id": "task-001"})

	if !result.IsError {
		t.Fatal("expected error on ACS failure")
	}
}

// ---------- NewRetryTask ----------

func TestRetryTask_returns_success(t *testing.T) {
	acs, cleanup := newTestACS(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/retry") {
			t.Errorf("expected /retry in path, got %s", r.URL.Path)
		}
		w.WriteHeader(200)
	})
	defer cleanup()

	_, handler := NewRetryTask(acs)
	result := callTool(t, handler, map[string]any{"task_id": "task-001"})

	if result.IsError {
		t.Fatalf("expected success, got error: %s", resultText(t, result))
	}
	text := resultText(t, result)
	if !strings.Contains(text, "task-001") {
		t.Errorf("expected task ID in response, got: %s", text)
	}
	if !strings.Contains(text, "retried") {
		t.Errorf("expected 'retried' in response, got: %s", text)
	}
}

func TestRetryTask_returns_error_when_task_id_missing(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`{}`))
	defer cleanup()

	_, handler := NewRetryTask(acs)
	result := callTool(t, handler, map[string]any{})

	if !result.IsError {
		t.Fatal("expected error when task_id missing")
	}
}

func TestRetryTask_returns_error_when_ACS_fails(t *testing.T) {
	acs, cleanup := newTestACS(t, errHandler(500, "internal error"))
	defer cleanup()

	_, handler := NewRetryTask(acs)
	result := callTool(t, handler, map[string]any{"task_id": "task-001"})

	if !result.IsError {
		t.Fatal("expected error on ACS failure")
	}
}

// ---------- Tool definition tests (verify tool metadata) ----------

func TestNewReboot_returns_tool_with_correct_name(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`{}`))
	defer cleanup()

	tool, _ := NewReboot(acs)
	if tool.Name != "reboot_device" {
		t.Errorf("expected tool name reboot_device, got %s", tool.Name)
	}
}

func TestNewDownloadFirmware_returns_tool_with_correct_name(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`{}`))
	defer cleanup()

	tool, _ := NewDownloadFirmware(acs)
	if tool.Name != "download_firmware" {
		t.Errorf("expected tool name download_firmware, got %s", tool.Name)
	}
}

func TestNewRefreshParameter_returns_tool_with_correct_name(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`{}`))
	defer cleanup()

	tool, _ := NewRefreshParameter(acs)
	if tool.Name != "refresh_parameter" {
		t.Errorf("expected tool name refresh_parameter, got %s", tool.Name)
	}
}

func TestNewSetParameter_returns_tool_with_correct_name(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`{}`))
	defer cleanup()

	tool, _ := NewSetParameter(acs)
	if tool.Name != "set_parameter" {
		t.Errorf("expected tool name set_parameter, got %s", tool.Name)
	}
}

func TestNewGetParameter_returns_tool_with_correct_name(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`{}`))
	defer cleanup()

	tool, _ := NewGetParameter(acs)
	if tool.Name != "get_parameter" {
		t.Errorf("expected tool name get_parameter, got %s", tool.Name)
	}
}

func TestNewManagePreset_returns_tool_with_correct_name(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`{}`))
	defer cleanup()

	tool, _ := NewManagePreset(acs)
	if tool.Name != "manage_preset" {
		t.Errorf("expected tool name manage_preset, got %s", tool.Name)
	}
}

func TestNewManageProvision_returns_tool_with_correct_name(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`{}`))
	defer cleanup()

	tool, _ := NewManageProvision(acs)
	if tool.Name != "manage_provision" {
		t.Errorf("expected tool name manage_provision, got %s", tool.Name)
	}
}

func TestNewSearchDevices_returns_tool_with_correct_name(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`{}`))
	defer cleanup()

	tool, _ := NewSearchDevices(acs)
	if tool.Name != "search_devices" {
		t.Errorf("expected tool name search_devices, got %s", tool.Name)
	}
}

func TestNewTagDevice_returns_tool_with_correct_name(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`{}`))
	defer cleanup()

	tool, _ := NewTagDevice(acs)
	if tool.Name != "tag_device" {
		t.Errorf("expected tool name tag_device, got %s", tool.Name)
	}
}

func TestNewConnectionRequest_returns_tool_with_correct_name(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`{}`))
	defer cleanup()

	tool, _ := NewConnectionRequest(acs)
	if tool.Name != "connection_request" {
		t.Errorf("expected tool name connection_request, got %s", tool.Name)
	}
}

func TestNewDeleteTask_returns_tool_with_correct_name(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`{}`))
	defer cleanup()

	tool, _ := NewDeleteTask(acs)
	if tool.Name != "delete_task" {
		t.Errorf("expected tool name delete_task, got %s", tool.Name)
	}
}

func TestNewRetryTask_returns_tool_with_correct_name(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`{}`))
	defer cleanup()

	tool, _ := NewRetryTask(acs)
	if tool.Name != "retry_task" {
		t.Errorf("expected tool name retry_task, got %s", tool.Name)
	}
}
