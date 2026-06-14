package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/geiserx/genieacs-mcp/client"
	"github.com/mark3labs/mcp-go/server"
)

func TestIsLoopbackAddr(t *testing.T) {
	cases := map[string]bool{
		"127.0.0.1:8080": true,
		"localhost:8080": true,
		"[::1]:8080":     true,
		"0.0.0.0:8080":   false,
		"192.168.1.5:80": false,
		"example.com:80": false,
		"garbage":        false,
	}
	for addr, want := range cases {
		if got := isLoopbackAddr(addr); got != want {
			t.Errorf("isLoopbackAddr(%q) = %v, want %v", addr, got, want)
		}
	}
}

func TestAllowedHostSet_LoopbackDefaults(t *testing.T) {
	hosts := allowedHostSet("127.0.0.1:8080", nil)
	for _, want := range []string{"127.0.0.1:8080", "localhost:8080", "[::1]:8080"} {
		if !hosts[want] {
			t.Errorf("expected host %q to be allowed; set=%v", want, hosts)
		}
	}
	if hosts["attacker.example:8080"] {
		t.Error("attacker host must not be allowed")
	}
}

func TestAllowedHostSet_WildcardBindStillTrustsLoopback(t *testing.T) {
	// Binding 0.0.0.0 must not add a literal "0.0.0.0:8080" Host entry, but
	// loopback names on the port stay trusted.
	hosts := allowedHostSet("0.0.0.0:8080", nil)
	if hosts["0.0.0.0:8080"] {
		t.Error("wildcard bind host should not be a trusted Host value")
	}
	if !hosts["127.0.0.1:8080"] {
		t.Error("loopback host should remain trusted under wildcard bind")
	}
}

func TestAllowedHostSet_Extra(t *testing.T) {
	hosts := allowedHostSet("127.0.0.1:8080", []string{"acs.example.com"})
	if !hosts["acs.example.com"] {
		t.Error("explicit extra host should be allowed")
	}
}

func TestAllowedOriginSet(t *testing.T) {
	hosts := allowedHostSet("127.0.0.1:8080", nil)
	origins := allowedOriginSet(hosts, []string{"https://app.example.com/"})
	for _, want := range []string{"http://127.0.0.1:8080", "https://localhost:8080"} {
		if !origins[want] {
			t.Errorf("expected origin %q to be derived from hosts", want)
		}
	}
	if !origins["https://app.example.com"] {
		t.Error("explicit origin should be allowed (trailing slash trimmed)")
	}
	if origins["http://attacker.example:8080"] {
		t.Error("attacker origin must not be allowed")
	}
}

func TestSplitList(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"   ", nil},
		{"a", []string{"a"}},
		{"a,b , c", []string{"a", "b", "c"}},
		{",,a,,", []string{"a"}},
	}
	for _, c := range cases {
		got := splitList(c.in)
		if len(got) != len(c.want) {
			t.Errorf("splitList(%q) = %v, want %v", c.in, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("splitList(%q)[%d] = %q, want %q", c.in, i, got[i], c.want[i])
			}
		}
	}
}

func TestBearerAuth(t *testing.T) {
	backend := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := bearerAuth(backend, "s3cret")

	t.Run("valid token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8080/mcp", nil)
		req.Header.Set("Authorization", "Bearer s3cret")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("valid token got %d, want 200", rec.Code)
		}
	})

	t.Run("wrong token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8080/mcp", nil)
		req.Header.Set("Authorization", "Bearer nope")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("wrong token got %d, want 401", rec.Code)
		}
	})

	t.Run("missing token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8080/mcp", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("missing token got %d, want 401", rec.Code)
		}
	})
}

func TestNewMCPServer(t *testing.T) {
	// Registering all resources and tools must not panic, and the server must
	// be usable as an HTTP handler backend.
	acs := client.NewACS("http://127.0.0.1:7557", "", "")
	s := newMCPServer(acs, 500)
	if s == nil {
		t.Fatal("newMCPServer returned nil")
	}
	httpSrv := server.NewStreamableHTTPServer(s)
	_, srv, err := newHTTPServer(httpSrv, httpEnv{addr: "127.0.0.1:8080"})
	if err != nil || srv == nil {
		t.Fatalf("server build failed: %v", err)
	}
}

func TestRunStdio_ReturnsOnEOF(t *testing.T) {
	acs := client.NewACS("http://127.0.0.1:7557", "", "")
	s := newMCPServer(acs, 500)
	// An empty reader is immediately at EOF, so Listen returns cleanly.
	done := make(chan error, 1)
	go func() { done <- runStdio(context.Background(), s, strings.NewReader(""), io.Discard) }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("runStdio returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("runStdio did not return on EOF")
	}
}

func TestRun_StdioTransport(t *testing.T) {
	t.Setenv("TRANSPORT", "stdio")
	acs := client.NewACS("http://127.0.0.1:7557", "", "")
	s := newMCPServer(acs, 500)
	done := make(chan error, 1)
	go func() { done <- run(context.Background(), s, strings.NewReader(""), io.Discard) }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run(stdio) returned error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("run(stdio) did not return on EOF")
	}
}

func TestRun_HTTPTransport_ShutsDownOnContextCancel(t *testing.T) {
	t.Setenv("TRANSPORT", "")
	t.Setenv("MCP_LISTEN_ADDR", "127.0.0.1:0") // ephemeral port, avoids clashes
	t.Setenv("MCP_AUTH_TOKEN", "")
	acs := client.NewACS("http://127.0.0.1:7557", "", "")
	s := newMCPServer(acs, 500)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- run(ctx, s, nil, nil) }()

	// Give the listener a moment to start, then cancel for graceful shutdown.
	time.Sleep(200 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run(http) returned error on shutdown: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("run(http) did not shut down on context cancel")
	}
}

func TestRun_HTTPTransport_AuthRequiredError(t *testing.T) {
	t.Setenv("TRANSPORT", "")
	t.Setenv("MCP_LISTEN_ADDR", "0.0.0.0:0")
	t.Setenv("MCP_AUTH_TOKEN", "")
	acs := client.NewACS("http://127.0.0.1:7557", "", "")
	s := newMCPServer(acs, 500)
	err := run(context.Background(), s, nil, nil)
	if !errors.Is(err, errAuthTokenRequired) {
		t.Fatalf("run() err = %v, want errAuthTokenRequired", err)
	}
}

func TestServeHTTP_GracefulShutdown(t *testing.T) {
	srv := &http.Server{
		Addr: "127.0.0.1:0",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- serveHTTP(ctx, srv) }()
	time.Sleep(150 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("serveHTTP shutdown error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("serveHTTP did not return after cancel")
	}
}

func TestServeHTTP_ListenError(t *testing.T) {
	// An invalid address makes ListenAndServe fail immediately; serveHTTP must
	// surface that error rather than hang.
	srv := &http.Server{Addr: "256.256.256.256:99999"}
	err := serveHTTP(context.Background(), srv)
	if err == nil {
		t.Fatal("serveHTTP expected an error for an invalid listen address")
	}
}

func TestLoadHTTPEnv(t *testing.T) {
	t.Setenv("MCP_LISTEN_ADDR", "")
	t.Setenv("MCP_AUTH_TOKEN", "")
	t.Setenv("MCP_ALLOWED_HOSTS", "")
	t.Setenv("MCP_ALLOWED_ORIGINS", "")
	if env := loadHTTPEnv(); env.addr != "127.0.0.1:8080" {
		t.Errorf("default addr = %q, want 127.0.0.1:8080", env.addr)
	}

	t.Setenv("MCP_LISTEN_ADDR", "0.0.0.0:9000")
	t.Setenv("MCP_AUTH_TOKEN", "tok")
	t.Setenv("MCP_ALLOWED_HOSTS", "a.example")
	t.Setenv("MCP_ALLOWED_ORIGINS", "https://a.example")
	env := loadHTTPEnv()
	if env.addr != "0.0.0.0:9000" || env.authToken != "tok" ||
		env.allowedHosts != "a.example" || env.allowedOrigin != "https://a.example" {
		t.Errorf("loadHTTPEnv did not read all vars: %+v", env)
	}
}

func TestNewHTTPServer(t *testing.T) {
	backend := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("loopback without token is allowed and guards /mcp", func(t *testing.T) {
		addr, srv, err := newHTTPServer(backend, httpEnv{addr: "127.0.0.1:8080"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if addr != "127.0.0.1:8080" || srv == nil || srv.Addr != "127.0.0.1:8080" {
			t.Fatalf("unexpected addr/srv: addr=%q srv=%v", addr, srv)
		}
		// Legitimate loopback request passes the mounted guard.
		req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8080/mcp", nil)
		req.Host = "127.0.0.1:8080"
		rec := httptest.NewRecorder()
		srv.Handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("loopback /mcp got %d, want 200", rec.Code)
		}
		// Rebind request is rejected.
		req2 := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8080/mcp", nil)
		req2.Host = "attacker.example:8080"
		rec2 := httptest.NewRecorder()
		srv.Handler.ServeHTTP(rec2, req2)
		if rec2.Code != http.StatusForbidden {
			t.Errorf("rebind /mcp got %d, want 403", rec2.Code)
		}
	})

	t.Run("non-loopback without token errors", func(t *testing.T) {
		_, _, err := newHTTPServer(backend, httpEnv{addr: "0.0.0.0:8080"})
		if !errors.Is(err, errAuthTokenRequired) {
			t.Fatalf("err = %v, want errAuthTokenRequired", err)
		}
	})

	t.Run("non-loopback with token is allowed", func(t *testing.T) {
		_, srv, err := newHTTPServer(backend, httpEnv{addr: "0.0.0.0:8080", authToken: "tok", allowedHosts: "acs.example.com"})
		if err != nil || srv == nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

// newGuardedHandler builds the same handler chain as main() for a loopback
// listener: a 200-returning backend wrapped in the DNS-rebinding guard.
func newGuardedHandler(addr string) http.Handler {
	backend := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	hosts := allowedHostSet(addr, nil)
	origins := allowedOriginSet(hosts, nil)
	return dnsRebindGuard(backend, hosts, origins)
}

func TestDNSRebindGuard_AllowsLoopbackClient(t *testing.T) {
	h := newGuardedHandler("127.0.0.1:8080")
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8080/mcp", nil)
	req.Host = "127.0.0.1:8080"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("loopback client got %d, want 200", rec.Code)
	}
}

func TestDNSRebindGuard_AllowsNonBrowserClientWithoutOrigin(t *testing.T) {
	// Trusted Host, no Origin header (typical CLI / LLM MCP client).
	h := newGuardedHandler("127.0.0.1:8080")
	req := httptest.NewRequest(http.MethodPost, "http://localhost:8080/mcp", nil)
	req.Host = "localhost:8080"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("non-browser client got %d, want 200", rec.Code)
	}
}

func TestDNSRebindGuard_BlocksRebindHost(t *testing.T) {
	// The exact PoC shape from GHSA-cmwv-wf9p-p8wx: attacker Host + Origin.
	h := newGuardedHandler("127.0.0.1:8080")
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8080/mcp", nil)
	req.Host = "attacker.example:8080"
	req.Header.Set("Origin", "http://attacker.example:8080")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("rebind request got %d, want 403", rec.Code)
	}
}

func TestDNSRebindGuard_BlocksUntrustedOriginOnTrustedHost(t *testing.T) {
	// Host looks local but the browser Origin is the attacker's page.
	h := newGuardedHandler("127.0.0.1:8080")
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8080/mcp", nil)
	req.Host = "127.0.0.1:8080"
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("untrusted-origin request got %d, want 403", rec.Code)
	}
}

func TestBuildHTTPHandler(t *testing.T) {
	backend := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// request is a helper that runs one request through h.
	request := func(h http.Handler, host, origin, auth string) int {
		req := httptest.NewRequest(http.MethodPost, "http://"+host+"/mcp", nil)
		req.Host = host
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		if auth != "" {
			req.Header.Set("Authorization", auth)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}

	t.Run("no auth: guard still enforced", func(t *testing.T) {
		h := buildHTTPHandler(backend, "127.0.0.1:8080", "", "", "")
		if code := request(h, "127.0.0.1:8080", "", ""); code != http.StatusOK {
			t.Errorf("loopback request got %d, want 200", code)
		}
		if code := request(h, "attacker.example:8080", "http://attacker.example:8080", ""); code != http.StatusForbidden {
			t.Errorf("rebind request got %d, want 403", code)
		}
	})

	t.Run("auth: guard runs and bearer required", func(t *testing.T) {
		h := buildHTTPHandler(backend, "127.0.0.1:8080", "tok", "", "")
		// Trusted host but missing token -> 401.
		if code := request(h, "127.0.0.1:8080", "", ""); code != http.StatusUnauthorized {
			t.Errorf("missing token got %d, want 401", code)
		}
		// Trusted host with valid token -> 200.
		if code := request(h, "127.0.0.1:8080", "", "Bearer tok"); code != http.StatusOK {
			t.Errorf("valid token got %d, want 200", code)
		}
		// Untrusted host is rejected before auth is even checked -> 403.
		if code := request(h, "attacker.example:8080", "", "Bearer tok"); code != http.StatusForbidden {
			t.Errorf("untrusted host got %d, want 403", code)
		}
	})

	t.Run("extra allowlists honoured", func(t *testing.T) {
		h := buildHTTPHandler(backend, "127.0.0.1:8080", "", "acs.example.com", "https://app.example.com")
		if code := request(h, "acs.example.com", "https://app.example.com", ""); code != http.StatusOK {
			t.Errorf("allowlisted host/origin got %d, want 200", code)
		}
	})
}
