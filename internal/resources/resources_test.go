package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/geiserx/genieacs-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ---------- helpers ----------

func newTestACS(t *testing.T, handler http.HandlerFunc) (*client.ACSClient, func()) {
	t.Helper()
	srv := httptest.NewServer(handler)
	acs := client.NewACS(srv.URL, "", "")
	return acs, srv.Close
}

func okHandler(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(body))
	}
}

func errHandler(code int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(code)
		w.Write([]byte(body))
	}
}

func newMCPServer() *server.MCPServer {
	return server.NewMCPServer("test", "0.0.0", server.WithRecovery())
}

// readResource sends a resources/read JSON-RPC request through HandleMessage
// and returns the result text or an error description.
func readResource(t *testing.T, s *server.MCPServer, uri string) (string, bool) {
	t.Helper()
	msg := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"resources/read","params":{"uri":%q}}`, uri)
	resp := s.HandleMessage(context.Background(), json.RawMessage(msg))

	// Check for JSON-RPC error
	if errResp, ok := resp.(mcp.JSONRPCError); ok {
		return errResp.Error.Message, true
	}

	// Successful response
	jsonResp, ok := resp.(mcp.JSONRPCResponse)
	if !ok {
		t.Fatalf("expected JSONRPCResponse, got %T", resp)
	}

	result, ok := jsonResp.Result.(mcp.ReadResourceResult)
	if !ok {
		t.Fatalf("expected ReadResourceResult, got %T", jsonResp.Result)
	}

	if len(result.Contents) == 0 {
		return "", false
	}

	trc, ok := result.Contents[0].(mcp.TextResourceContents)
	if !ok {
		t.Fatalf("expected TextResourceContents, got %T", result.Contents[0])
	}
	return trc.Text, false
}

// ---------- RegisterDevice ----------

func TestRegisterDevice_returns_device_JSON_for_valid_ID(t *testing.T) {
	acs, cleanup := newTestACS(t, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		if !strings.Contains(query, "dev-001") {
			t.Errorf("expected query to contain dev-001, got %s", query)
		}
		w.Write([]byte(`[{"_id":"dev-001","_tags":["office"]}]`))
	})
	defer cleanup()

	s := newMCPServer()
	RegisterDevice(s, acs)

	text, isErr := readResource(t, s, "genieacs://device/dev-001")
	if isErr {
		t.Fatalf("expected success, got error: %s", text)
	}
	if !strings.Contains(text, "dev-001") {
		t.Errorf("expected response to contain dev-001, got: %s", text)
	}
}

func TestRegisterDevice_returns_error_for_empty_ID(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`[]`))
	defer cleanup()

	s := newMCPServer()
	RegisterDevice(s, acs)

	_, isErr := readResource(t, s, "genieacs://device/")
	if !isErr {
		t.Fatal("expected error for empty device ID")
	}
}

func TestRegisterDevice_returns_error_when_ACS_fails(t *testing.T) {
	acs, cleanup := newTestACS(t, errHandler(500, "internal error"))
	defer cleanup()

	s := newMCPServer()
	RegisterDevice(s, acs)

	_, isErr := readResource(t, s, "genieacs://device/dev-001")
	if !isErr {
		t.Fatal("expected error when ACS returns 500")
	}
}

// ---------- RegisterFile ----------

func TestRegisterFile_returns_file_metadata_for_valid_name(t *testing.T) {
	acs, cleanup := newTestACS(t, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		if !strings.Contains(query, "firmware.bin") {
			t.Errorf("expected query to contain firmware.bin, got %s", query)
		}
		w.Write([]byte(`[{"filename":"firmware.bin","_id":"abc"}]`))
	})
	defer cleanup()

	s := newMCPServer()
	RegisterFile(s, acs)

	text, isErr := readResource(t, s, "genieacs://file/firmware.bin")
	if isErr {
		t.Fatalf("expected success, got error: %s", text)
	}
	if !strings.Contains(text, "firmware.bin") {
		t.Errorf("expected response to contain firmware.bin, got: %s", text)
	}
}

func TestRegisterFile_returns_error_for_empty_name(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`[]`))
	defer cleanup()

	s := newMCPServer()
	RegisterFile(s, acs)

	_, isErr := readResource(t, s, "genieacs://file/")
	if !isErr {
		t.Fatal("expected error for empty file name")
	}
}

func TestRegisterFile_returns_error_when_ACS_fails(t *testing.T) {
	acs, cleanup := newTestACS(t, errHandler(404, "not found"))
	defer cleanup()

	s := newMCPServer()
	RegisterFile(s, acs)

	_, isErr := readResource(t, s, "genieacs://file/missing.bin")
	if !isErr {
		t.Fatal("expected error when ACS returns 404")
	}
}

// ---------- RegisterTasks ----------

func TestRegisterTasks_returns_tasks_for_device(t *testing.T) {
	acs, cleanup := newTestACS(t, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		if !strings.Contains(query, "dev-001") {
			t.Errorf("expected query to contain dev-001, got %s", query)
		}
		w.Write([]byte(`[{"_id":"task-1","name":"reboot"}]`))
	})
	defer cleanup()

	s := newMCPServer()
	RegisterTasks(s, acs)

	text, isErr := readResource(t, s, "genieacs://tasks/dev-001")
	if isErr {
		t.Fatalf("expected success, got error: %s", text)
	}
	if !strings.Contains(text, "task-1") {
		t.Errorf("expected response to contain task-1, got: %s", text)
	}
}

func TestRegisterTasks_returns_error_for_empty_ID(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`[]`))
	defer cleanup()

	s := newMCPServer()
	RegisterTasks(s, acs)

	_, isErr := readResource(t, s, "genieacs://tasks/")
	if !isErr {
		t.Fatal("expected error for empty device ID")
	}
}

func TestRegisterTasks_returns_error_when_ACS_fails(t *testing.T) {
	acs, cleanup := newTestACS(t, errHandler(500, "db error"))
	defer cleanup()

	s := newMCPServer()
	RegisterTasks(s, acs)

	_, isErr := readResource(t, s, "genieacs://tasks/dev-001")
	if !isErr {
		t.Fatal("expected error when ACS returns 500")
	}
}

// ---------- RegisterCatalogue ----------

func TestRegisterCatalogue_returns_device_summaries(t *testing.T) {
	acs, cleanup := newTestACS(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/devices" {
			t.Errorf("expected /devices, got %s", r.URL.Path)
		}
		limit := r.URL.Query().Get("limit")
		if limit != "100" {
			t.Errorf("expected limit 100, got %s", limit)
		}
		w.Write([]byte(`[{"_id":"dev-001"}]`))
	})
	defer cleanup()

	s := newMCPServer()
	RegisterCatalogue(s, acs, 100)

	text, isErr := readResource(t, s, "genieacs://devices/list")
	if isErr {
		t.Fatalf("expected success, got error: %s", text)
	}
	if !strings.Contains(text, "dev-001") {
		t.Errorf("expected response to contain dev-001, got: %s", text)
	}
}

func TestRegisterCatalogue_returns_error_when_ACS_fails(t *testing.T) {
	acs, cleanup := newTestACS(t, errHandler(500, "overloaded"))
	defer cleanup()

	s := newMCPServer()
	RegisterCatalogue(s, acs, 500)

	_, isErr := readResource(t, s, "genieacs://devices/list")
	if !isErr {
		t.Fatal("expected error when ACS returns 500")
	}
}

// ---------- RegisterPresets ----------

func TestRegisterPresets_returns_preset_list(t *testing.T) {
	acs, cleanup := newTestACS(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/presets" {
			t.Errorf("expected /presets, got %s", r.URL.Path)
		}
		w.Write([]byte(`[{"_id":"preset-1","weight":0}]`))
	})
	defer cleanup()

	s := newMCPServer()
	RegisterPresets(s, acs)

	text, isErr := readResource(t, s, "genieacs://presets/list")
	if isErr {
		t.Fatalf("expected success, got error: %s", text)
	}
	if !strings.Contains(text, "preset-1") {
		t.Errorf("expected response to contain preset-1, got: %s", text)
	}
}

func TestRegisterPresets_returns_error_when_ACS_fails(t *testing.T) {
	acs, cleanup := newTestACS(t, errHandler(500, "db error"))
	defer cleanup()

	s := newMCPServer()
	RegisterPresets(s, acs)

	_, isErr := readResource(t, s, "genieacs://presets/list")
	if !isErr {
		t.Fatal("expected error when ACS returns 500")
	}
}

// ---------- RegisterProvisions ----------

func TestRegisterProvisions_returns_provision_list(t *testing.T) {
	acs, cleanup := newTestACS(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/provisions" {
			t.Errorf("expected /provisions, got %s", r.URL.Path)
		}
		w.Write([]byte(`[{"_id":"prov-1","script":"log('hi');"}]`))
	})
	defer cleanup()

	s := newMCPServer()
	RegisterProvisions(s, acs)

	text, isErr := readResource(t, s, "genieacs://provisions/list")
	if isErr {
		t.Fatalf("expected success, got error: %s", text)
	}
	if !strings.Contains(text, "prov-1") {
		t.Errorf("expected response to contain prov-1, got: %s", text)
	}
}

func TestRegisterProvisions_returns_error_when_ACS_fails(t *testing.T) {
	acs, cleanup := newTestACS(t, errHandler(500, "db error"))
	defer cleanup()

	s := newMCPServer()
	RegisterProvisions(s, acs)

	_, isErr := readResource(t, s, "genieacs://provisions/list")
	if !isErr {
		t.Fatal("expected error when ACS returns 500")
	}
}

// ---------- RegisterFaults ----------

func TestRegisterFaults_returns_faults_for_device(t *testing.T) {
	acs, cleanup := newTestACS(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/faults" {
			t.Errorf("expected /faults, got %s", r.URL.Path)
		}
		query := r.URL.Query().Get("query")
		if !strings.Contains(query, "dev-001") {
			t.Errorf("expected query to contain dev-001, got %s", query)
		}
		w.Write([]byte(`[{"_id":"dev-001:channel","code":"9002"}]`))
	})
	defer cleanup()

	s := newMCPServer()
	RegisterFaults(s, acs)

	text, isErr := readResource(t, s, "genieacs://faults/dev-001")
	if isErr {
		t.Fatalf("expected success, got error: %s", text)
	}
	if !strings.Contains(text, "9002") {
		t.Errorf("expected response to contain fault code 9002, got: %s", text)
	}
}

func TestRegisterFaults_returns_error_for_empty_ID(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`[]`))
	defer cleanup()

	s := newMCPServer()
	RegisterFaults(s, acs)

	_, isErr := readResource(t, s, "genieacs://faults/")
	if !isErr {
		t.Fatal("expected error for empty device ID")
	}
}

func TestRegisterFaults_returns_error_when_ACS_fails(t *testing.T) {
	acs, cleanup := newTestACS(t, errHandler(500, "db error"))
	defer cleanup()

	s := newMCPServer()
	RegisterFaults(s, acs)

	_, isErr := readResource(t, s, "genieacs://faults/dev-001")
	if !isErr {
		t.Fatal("expected error when ACS returns 500")
	}
}

// ---------- Registration does not panic ----------

func TestAllRegistrations_do_not_panic(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`[]`))
	defer cleanup()

	s := newMCPServer()

	RegisterDevice(s, acs)
	RegisterFile(s, acs)
	RegisterTasks(s, acs)
	RegisterCatalogue(s, acs, 500)
	RegisterPresets(s, acs)
	RegisterProvisions(s, acs)
	RegisterFaults(s, acs)
}

// ---------- URI and MIME type verification ----------

func TestDeviceHandler_returns_correct_URI_and_MIME(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`[{"_id":"dev-xyz"}]`))
	defer cleanup()

	s := newMCPServer()
	RegisterDevice(s, acs)

	msg := `{"jsonrpc":"2.0","id":1,"method":"resources/read","params":{"uri":"genieacs://device/dev-xyz"}}`
	resp := s.HandleMessage(context.Background(), json.RawMessage(msg))

	jsonResp := resp.(mcp.JSONRPCResponse)
	result := jsonResp.Result.(mcp.ReadResourceResult)
	trc := result.Contents[0].(mcp.TextResourceContents)

	if trc.URI != "genieacs://device/dev-xyz" {
		t.Errorf("expected URI genieacs://device/dev-xyz, got %s", trc.URI)
	}
	if trc.MIMEType != "application/json" {
		t.Errorf("expected MIME application/json, got %s", trc.MIMEType)
	}
}

func TestFileHandler_returns_correct_URI(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`[{"filename":"fw.bin"}]`))
	defer cleanup()

	s := newMCPServer()
	RegisterFile(s, acs)

	msg := `{"jsonrpc":"2.0","id":1,"method":"resources/read","params":{"uri":"genieacs://file/fw.bin"}}`
	resp := s.HandleMessage(context.Background(), json.RawMessage(msg))

	jsonResp := resp.(mcp.JSONRPCResponse)
	result := jsonResp.Result.(mcp.ReadResourceResult)
	trc := result.Contents[0].(mcp.TextResourceContents)

	if trc.URI != "genieacs://file/fw.bin" {
		t.Errorf("expected URI genieacs://file/fw.bin, got %s", trc.URI)
	}
}

func TestTasksHandler_returns_correct_URI(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`[{"_id":"t1"}]`))
	defer cleanup()

	s := newMCPServer()
	RegisterTasks(s, acs)

	msg := `{"jsonrpc":"2.0","id":1,"method":"resources/read","params":{"uri":"genieacs://tasks/dev-001"}}`
	resp := s.HandleMessage(context.Background(), json.RawMessage(msg))

	jsonResp := resp.(mcp.JSONRPCResponse)
	result := jsonResp.Result.(mcp.ReadResourceResult)
	trc := result.Contents[0].(mcp.TextResourceContents)

	if trc.URI != "genieacs://tasks/dev-001" {
		t.Errorf("expected URI genieacs://tasks/dev-001, got %s", trc.URI)
	}
}

func TestCatalogueHandler_returns_correct_URI(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`[{"_id":"dev-001"}]`))
	defer cleanup()

	s := newMCPServer()
	RegisterCatalogue(s, acs, 500)

	msg := `{"jsonrpc":"2.0","id":1,"method":"resources/read","params":{"uri":"genieacs://devices/list"}}`
	resp := s.HandleMessage(context.Background(), json.RawMessage(msg))

	jsonResp := resp.(mcp.JSONRPCResponse)
	result := jsonResp.Result.(mcp.ReadResourceResult)
	trc := result.Contents[0].(mcp.TextResourceContents)

	if trc.URI != "genieacs://devices/list" {
		t.Errorf("expected URI genieacs://devices/list, got %s", trc.URI)
	}
}

func TestFaultsHandler_returns_correct_URI(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`[]`))
	defer cleanup()

	s := newMCPServer()
	RegisterFaults(s, acs)

	msg := `{"jsonrpc":"2.0","id":1,"method":"resources/read","params":{"uri":"genieacs://faults/dev-001"}}`
	resp := s.HandleMessage(context.Background(), json.RawMessage(msg))

	jsonResp := resp.(mcp.JSONRPCResponse)
	result := jsonResp.Result.(mcp.ReadResourceResult)
	trc := result.Contents[0].(mcp.TextResourceContents)

	if trc.URI != "genieacs://faults/dev-001" {
		t.Errorf("expected URI genieacs://faults/dev-001, got %s", trc.URI)
	}
}

func TestPresetsHandler_returns_correct_URI(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`[{"_id":"p1"}]`))
	defer cleanup()

	s := newMCPServer()
	RegisterPresets(s, acs)

	msg := `{"jsonrpc":"2.0","id":1,"method":"resources/read","params":{"uri":"genieacs://presets/list"}}`
	resp := s.HandleMessage(context.Background(), json.RawMessage(msg))

	jsonResp := resp.(mcp.JSONRPCResponse)
	result := jsonResp.Result.(mcp.ReadResourceResult)
	trc := result.Contents[0].(mcp.TextResourceContents)

	if trc.URI != "genieacs://presets/list" {
		t.Errorf("expected URI genieacs://presets/list, got %s", trc.URI)
	}
}

func TestProvisionsHandler_returns_correct_URI(t *testing.T) {
	acs, cleanup := newTestACS(t, okHandler(`[{"_id":"pv1"}]`))
	defer cleanup()

	s := newMCPServer()
	RegisterProvisions(s, acs)

	msg := `{"jsonrpc":"2.0","id":1,"method":"resources/read","params":{"uri":"genieacs://provisions/list"}}`
	resp := s.HandleMessage(context.Background(), json.RawMessage(msg))

	jsonResp := resp.(mcp.JSONRPCResponse)
	result := jsonResp.Result.(mcp.ReadResourceResult)
	trc := result.Contents[0].(mcp.TextResourceContents)

	if trc.URI != "genieacs://provisions/list" {
		t.Errorf("expected URI genieacs://provisions/list, got %s", trc.URI)
	}
}
