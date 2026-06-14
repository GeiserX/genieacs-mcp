package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
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
