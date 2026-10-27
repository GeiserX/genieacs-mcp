package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/geiserx/genieacs-mcp/client"
)

// The protocol smoke tests drive the real server, built by newMCPServer, over
// the real transports. They check that both protocol eras work from one
// binary: 2026-07-28 clients, which send server/discover and carry the
// protocol version in every request's _meta, and older clients that start
// with initialize (Claude Code, Claude Desktop, Cursor and VS Code still do).

const modernVersion = "2026-07-28"

// modernMeta is the _meta every 2026-07-28 request carries in place of the
// initialize handshake.
func modernMeta() map[string]any {
	return map[string]any{
		"io.modelcontextprotocol/protocolVersion":    modernVersion,
		"io.modelcontextprotocol/clientCapabilities": map[string]any{},
		"io.modelcontextprotocol/clientInfo":         map[string]any{"name": "smoke", "version": "0"},
	}
}

func rpc(id int, method string, params map[string]any) string {
	msg := map[string]any{"jsonrpc": "2.0", "method": method}
	if id > 0 {
		msg["id"] = id
	}
	if params != nil {
		msg["params"] = params
	}
	b, _ := json.Marshal(msg)
	return string(b)
}

// stdioSession runs the server over the stdio transport in-process and
// returns a function that sends one line and waits for the reply to id.
func stdioSession(t *testing.T, acsURL ...string) (send func(line string, id int) map[string]any, notify func(line string)) {
	t.Helper()
	base := "http://127.0.0.1:7557"
	if len(acsURL) > 0 {
		base = acsURL[0]
	}
	acs := client.NewACS(base, "", "")
	s, err := newMCPServer(acs, serverOptions{deviceLimit: 500})
	if err != nil {
		t.Fatalf("newMCPServer: %v", err)
	}
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_ = runStdio(ctx, s, inR, outW)
		_ = outW.Close()
		close(done)
	}()
	t.Cleanup(func() {
		_ = inW.Close()
		cancel()
		<-done
	})
	lines := make(chan string, 16)
	go func() {
		sc := bufio.NewScanner(outR)
		sc.Buffer(make([]byte, 0, 1<<20), 8<<20)
		for sc.Scan() {
			lines <- sc.Text()
		}
		close(lines)
	}()
	notify = func(line string) {
		if _, err := io.WriteString(inW, line+"\n"); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	send = func(line string, id int) map[string]any {
		t.Helper()
		notify(line)
		deadline := time.After(5 * time.Second)
		for {
			select {
			case l, ok := <-lines:
				if !ok {
					t.Fatalf("server closed stdout before answering id %d", id)
				}
				var m map[string]any
				if err := json.Unmarshal([]byte(l), &m); err != nil {
					t.Fatalf("stdout carried a non-JSON line %q: %v", l, err)
				}
				if got, ok := m["id"].(float64); ok && int(got) == id {
					return m
				}
			case <-deadline:
				t.Fatalf("no reply to id %d within 5s", id)
			}
		}
	}
	return send, notify
}

func resultOf(t *testing.T, m map[string]any) map[string]any {
	t.Helper()
	if e, ok := m["error"]; ok {
		t.Fatalf("got JSON-RPC error %v, want a result", e)
	}
	r, ok := m["result"].(map[string]any)
	if !ok {
		t.Fatalf("reply has no result object: %v", m)
	}
	return r
}

func toolNames(t *testing.T, result map[string]any) []string {
	t.Helper()
	list, ok := result["tools"].([]any)
	if !ok {
		t.Fatalf("result has no tools array: %v", result)
	}
	var names []string
	for _, item := range list {
		names = append(names, item.(map[string]any)["name"].(string))
	}
	return names
}

func TestProtocol_Stdio_ModernEra(t *testing.T) {
	send, _ := stdioSession(t)

	// server/discover lists 2026-07-28 among the versions it serves, and the
	// legacy ones too, so a dual-era client can pick either.
	disc := resultOf(t, send(rpc(1, "server/discover", map[string]any{"_meta": modernMeta()}), 1))
	versions, _ := disc["supportedVersions"].([]any)
	for _, want := range []string{modernVersion, "2025-11-25", "2024-11-05"} {
		if !slices.Contains(versions, any(want)) {
			t.Errorf("server/discover supportedVersions = %v, want it to contain %s", versions, want)
		}
	}

	// A modern tools/list is a complete, cacheable result.
	list := resultOf(t, send(rpc(2, "tools/list", map[string]any{"_meta": modernMeta()}), 2))
	if list["resultType"] != "complete" {
		t.Errorf("tools/list resultType = %v, want complete", list["resultType"])
	}
	if _, ok := list["ttlMs"].(float64); !ok {
		t.Errorf("tools/list has no numeric ttlMs: %v", list["ttlMs"])
	}
	if scope, _ := list["cacheScope"].(string); scope != "public" && scope != "private" {
		t.Errorf("tools/list cacheScope = %v, want public or private", list["cacheScope"])
	}
	if names := toolNames(t, list); !slices.Contains(names, "search_devices") {
		t.Errorf("tools/list names = %v, want search_devices among them", names)
	}
	// The list is static, so clients may cache it.
	if list["ttlMs"] != float64(listCacheTTL) || list["cacheScope"] != "public" {
		t.Errorf("tools/list cache hints = %v/%v, want %d/public", list["ttlMs"], list["cacheScope"], listCacheTTL)
	}
	for _, item := range list["tools"].([]any) {
		tool := item.(map[string]any)
		if tool["name"] != "search_devices" {
			continue
		}
		ann, _ := tool["annotations"].(map[string]any)
		if ann["readOnlyHint"] != true || ann["destructiveHint"] != false {
			t.Errorf("search_devices annotations on the wire = %v, want read-only and not destructive", ann)
		}
		if tool["title"] == nil || tool["outputSchema"] == nil {
			t.Errorf("search_devices has no title or outputSchema on the wire: %v", tool)
		}
	}
	if disc["instructions"] == nil || disc["instructions"] == "" {
		t.Errorf("server/discover carries no instructions")
	}

	// ping was removed in 2026-07-28: a modern ping is method-not-found.
	ping := send(rpc(3, "ping", map[string]any{"_meta": modernMeta()}), 3)
	e, ok := ping["error"].(map[string]any)
	if !ok || e["code"] != float64(-32601) {
		t.Errorf("modern ping = %v, want error -32601", ping)
	}
}

func TestProtocol_Stdio_LegacyEra(t *testing.T) {
	send, notify := stdioSession(t)

	init := resultOf(t, send(rpc(1, "initialize", map[string]any{
		"protocolVersion": "2025-11-25",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "legacy", "version": "0"},
	}), 1))
	if init["protocolVersion"] != "2025-11-25" {
		t.Errorf("initialize protocolVersion = %v, want 2025-11-25", init["protocolVersion"])
	}
	notify(rpc(0, "notifications/initialized", nil))

	list := resultOf(t, send(rpc(2, "tools/list", nil), 2))
	if names := toolNames(t, list); len(names) != 12 {
		t.Errorf("legacy tools/list returned %d tools (%v), want 12", len(names), names)
	}
	// Legacy results stay in the old shape: no 2026-07-28 fields.
	if _, ok := list["resultType"]; ok {
		t.Errorf("legacy tools/list carries resultType %v; legacy clients must get the old shape", list["resultType"])
	}
}

// httpTestServer serves the server over Streamable HTTP through the same
// handler chain run() builds, on an ephemeral loopback port.
func httpTestServer(t *testing.T, env httpEnv) *httptest.Server {
	t.Helper()
	acs := client.NewACS("http://127.0.0.1:7557", "", "")
	s, err := newMCPServer(acs, serverOptions{deviceLimit: 500})
	if err != nil {
		t.Fatalf("newMCPServer: %v", err)
	}
	ts := httptest.NewUnstartedServer(nil)
	env.addr = ts.Listener.Addr().String()
	_, srv, err := newHTTPServer(newStreamableHandler(s), env)
	if err != nil {
		t.Fatalf("newHTTPServer: %v", err)
	}
	ts.Config = srv
	ts.Start()
	t.Cleanup(ts.Close)
	return ts
}

func postMCP(t *testing.T, url, body string, headers map[string]string) (int, map[string]any) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	// A reply may come as plain JSON or as one SSE "data:" event.
	if i := bytes.Index(raw, []byte("data: ")); i >= 0 && !bytes.HasPrefix(bytes.TrimSpace(raw), []byte("{")) {
		raw = raw[i+len("data: "):]
		if j := bytes.IndexByte(raw, '\n'); j >= 0 {
			raw = raw[:j]
		}
	}
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	return resp.StatusCode, m
}

func TestProtocol_HTTP_BothEras(t *testing.T) {
	ts := httpTestServer(t, httpEnv{})
	url := ts.URL + "/mcp"
	modern := func(method string) map[string]string {
		return map[string]string{"MCP-Protocol-Version": modernVersion, "Mcp-Method": method}
	}

	code, disc := postMCP(t, url, rpc(1, "server/discover", map[string]any{"_meta": modernMeta()}), modern("server/discover"))
	if code != http.StatusOK {
		t.Fatalf("modern server/discover over HTTP got %d: %v", code, disc)
	}
	versions, _ := resultOf(t, disc)["supportedVersions"].([]any)
	if !slices.Contains(versions, any(modernVersion)) {
		t.Errorf("HTTP server/discover supportedVersions = %v, want %s among them", versions, modernVersion)
	}

	code, list := postMCP(t, url, rpc(2, "tools/list", map[string]any{"_meta": modernMeta()}), modern("tools/list"))
	if code != http.StatusOK || resultOf(t, list)["resultType"] != "complete" {
		t.Errorf("modern tools/list over HTTP got %d: %v", code, list)
	}

	// A legacy client: initialize, then tools/list, with no session id kept
	// (the transport is stateless).
	code, init := postMCP(t, url, rpc(3, "initialize", map[string]any{
		"protocolVersion": "2025-06-18",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "legacy", "version": "0"},
	}), nil)
	if code != http.StatusOK || resultOf(t, init)["protocolVersion"] != "2025-06-18" {
		t.Fatalf("legacy initialize over HTTP got %d: %v", code, init)
	}
	code, legacyList := postMCP(t, url, rpc(4, "tools/list", nil), map[string]string{"MCP-Protocol-Version": "2025-06-18"})
	if code != http.StatusOK {
		t.Fatalf("legacy tools/list over HTTP got %d: %v", code, legacyList)
	}
	if n := len(toolNames(t, resultOf(t, legacyList))); n != 12 {
		t.Errorf("legacy tools/list over HTTP returned %d tools, want 12", n)
	}
}

func TestProtocol_HTTP_StatelessNoSessionID(t *testing.T) {
	ts := httpTestServer(t, httpEnv{})
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/mcp", strings.NewReader(rpc(1, "initialize", map[string]any{
		"protocolVersion": "2025-11-25",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "legacy", "version": "0"},
	})))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if id := resp.Header.Get("Mcp-Session-Id"); id != "" {
		t.Errorf("stateless server minted session id %q; sessions grow without bound", id)
	}
}

// Structured results pass the server's own output-schema validation, which
// turns a result that breaks its outputSchema into an error.
func TestProtocol_Stdio_StructuredToolResults(t *testing.T) {
	nbi := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost:
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"_id":"task-1","name":"reboot"}`))
		case r.URL.Query().Get("query") == `{"_id":"dev-1"}`:
			_, _ = w.Write([]byte(`[{"_id":"dev-1","Device":{"DeviceInfo":{"SoftwareVersion":{"_value":"1.0","_type":"xsd:string"}}}}]`))
		default:
			_, _ = w.Write([]byte(`[{"_id":"dev-1","_tags":["office"]}]`))
		}
	}))
	defer nbi.Close()
	send, _ := stdioSession(t, nbi.URL)

	calls := []struct {
		name string
		args map[string]any
		key  string
	}{
		{"search_devices", map[string]any{"query": `{"_tags":"office"}`, "limit": 10}, "devices"},
		{"get_parameter", map[string]any{"device_id": "dev-1", "parameter_path": "Device.DeviceInfo.SoftwareVersion"}, "parameters"},
		{"reboot_device", map[string]any{"device_id": "dev-1"}, "status"},
	}
	for i, c := range calls {
		params := map[string]any{"_meta": modernMeta(), "name": c.name, "arguments": c.args}
		res := resultOf(t, send(rpc(10+i, "tools/call", params), 10+i))
		if res["isError"] == true {
			t.Errorf("%s returned an error: %v", c.name, res["content"])
			continue
		}
		sc, ok := res["structuredContent"].(map[string]any)
		if !ok || sc[c.key] == nil {
			t.Errorf("%s structuredContent = %v, want a %q field", c.name, res["structuredContent"], c.key)
		}
	}

	// Input validation: an unknown argument and an out-of-range limit are
	// refused before the handler runs.
	for i, args := range []map[string]any{
		{"query": `{}`, "limit": 1000},
		{"query": `{}`, "bogus": true},
	} {
		params := map[string]any{"_meta": modernMeta(), "name": "search_devices", "arguments": args}
		if res := resultOf(t, send(rpc(20+i, "tools/call", params), 20+i)); res["isError"] != true {
			t.Errorf("search_devices accepted invalid arguments %v", args)
		}
	}
}
