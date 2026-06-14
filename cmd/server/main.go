package main

import (
	"context"
	"crypto/subtle"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/geiserx/genieacs-mcp/client"
	"github.com/geiserx/genieacs-mcp/config"
	"github.com/geiserx/genieacs-mcp/internal/resources"
	"github.com/geiserx/genieacs-mcp/internal/tools"
	"github.com/geiserx/genieacs-mcp/version"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func isLoopbackAddr(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	if host == "" || host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func isWildcardHost(host string) bool {
	switch host {
	case "", "0.0.0.0", "::", "[::]":
		return true
	}
	return false
}

// splitList parses a comma-separated environment value into a trimmed,
// non-empty slice.
func splitList(v string) []string {
	var out []string
	for _, item := range strings.Split(v, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}

// allowedHostSet builds the set of acceptable Host header values for a
// listener on addr. Loopback names on the listen port are always trusted so
// the documented local setup works with zero configuration. A concrete
// (non-wildcard) bind host is trusted as well, and operators can add extra
// names — e.g. a reverse-proxy domain — via MCP_ALLOWED_HOSTS.
func allowedHostSet(addr string, extra []string) map[string]bool {
	set := map[string]bool{}
	host, port, err := net.SplitHostPort(addr)
	if err == nil && port != "" {
		for _, h := range []string{"127.0.0.1", "localhost", "::1"} {
			set[net.JoinHostPort(h, port)] = true
		}
		if !isWildcardHost(host) {
			set[net.JoinHostPort(host, port)] = true
		}
	}
	for _, h := range extra {
		set[h] = true
	}
	return set
}

// allowedOriginSet derives acceptable browser Origin values (http/https) from
// the allowed hosts, plus any explicit MCP_ALLOWED_ORIGINS entries.
func allowedOriginSet(hosts map[string]bool, extra []string) map[string]bool {
	set := map[string]bool{}
	for h := range hosts {
		set["http://"+h] = true
		set["https://"+h] = true
	}
	for _, o := range extra {
		set[strings.TrimRight(o, "/")] = true
	}
	return set
}

// dnsRebindGuard rejects requests whose Host header is not in the allowed set,
// and requests carrying an Origin that is not allowed. This blocks DNS
// rebinding: after a rebind the browser still sends the attacker's Host/Origin
// (e.g. "attacker.example:8080"), which is not trusted. A missing Origin is
// permitted because non-browser MCP clients do not send one; the Host check is
// what stops the browser-driven attack.
func dnsRebindGuard(next http.Handler, hosts, origins map[string]bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !hosts[r.Host] {
			http.Error(w, "forbidden: untrusted Host header", http.StatusForbidden)
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" && !origins[strings.TrimRight(origin, "/")] {
			http.Error(w, "forbidden: untrusted Origin header", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// buildHTTPHandler wraps the MCP handler for the HTTP transport: an outer
// DNS-rebinding guard (Host/Origin validation derived from addr plus the
// MCP_ALLOWED_HOSTS / MCP_ALLOWED_ORIGINS allowlists) and, when authToken is
// set, an inner bearer-auth check.
func buildHTTPHandler(mcpHandler http.Handler, addr, authToken, extraHosts, extraOrigins string) http.Handler {
	allowedHosts := allowedHostSet(addr, splitList(extraHosts))
	allowedOrigins := allowedOriginSet(allowedHosts, splitList(extraOrigins))

	handler := mcpHandler
	if authToken != "" {
		handler = bearerAuth(handler, authToken)
	}
	return dnsRebindGuard(handler, allowedHosts, allowedOrigins)
}

// httpEnv is the subset of environment configuration the HTTP transport needs.
type httpEnv struct {
	addr          string
	authToken     string
	allowedHosts  string
	allowedOrigin string
}

// loadHTTPEnv reads the HTTP-transport settings from the environment, applying
// the loopback default for the listen address.
func loadHTTPEnv() httpEnv {
	addr := os.Getenv("MCP_LISTEN_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8080"
	}
	return httpEnv{
		addr:          addr,
		authToken:     os.Getenv("MCP_AUTH_TOKEN"),
		allowedHosts:  os.Getenv("MCP_ALLOWED_HOSTS"),
		allowedOrigin: os.Getenv("MCP_ALLOWED_ORIGINS"),
	}
}

// newHTTPServer builds the listening address and *http.Server for the HTTP
// transport, mounting the guarded MCP handler at /mcp. It returns an error
// when an auth token is required (non-loopback listener) but absent, so the
// caller can decide how to fail. The returned server is not yet listening.
func newHTTPServer(mcpHandler http.Handler, env httpEnv) (string, *http.Server, error) {
	if env.authToken == "" && !isLoopbackAddr(env.addr) {
		return "", nil, errAuthTokenRequired
	}

	// Validate Host/Origin on every request to prevent DNS rebinding from a
	// malicious web page reaching this listener (GHSA-cmwv-wf9p-p8wx).
	handler := buildHTTPHandler(mcpHandler, env.addr, env.authToken, env.allowedHosts, env.allowedOrigin)

	mux := http.NewServeMux()
	mux.Handle("/mcp", handler)
	return env.addr, &http.Server{Addr: env.addr, Handler: mux}, nil
}

var errAuthTokenRequired = errors.New("MCP_AUTH_TOKEN is required when MCP_LISTEN_ADDR is not loopback")

func bearerAuth(next http.Handler, token string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// newMCPServer builds the MCP server with all GenieACS resources and tools
// registered against acs.
func newMCPServer(acs *client.ACSClient, deviceLimit int) *server.MCPServer {
	s := server.NewMCPServer(
		"GenieACS MCP Bridge",
		version.Version,
		server.WithToolCapabilities(true),
		server.WithRecovery(),
	)

	// --- Resources ---
	resources.RegisterDevice(s, acs)
	resources.RegisterFile(s, acs)
	resources.RegisterTasks(s, acs)
	resources.RegisterCatalogue(s, acs, deviceLimit)
	resources.RegisterPresets(s, acs)
	resources.RegisterProvisions(s, acs)
	resources.RegisterFaults(s, acs)

	// --- Tools ---
	register := func(factory func(*client.ACSClient) (mcp.Tool, server.ToolHandlerFunc)) {
		t, h := factory(acs)
		s.AddTool(t, h)
	}
	register(tools.NewReboot)
	register(tools.NewDownloadFirmware)
	register(tools.NewRefreshParameter)
	register(tools.NewSetParameter)
	register(tools.NewGetParameter)
	register(tools.NewManagePreset)
	register(tools.NewManageProvision)
	register(tools.NewSearchDevices)
	register(tools.NewTagDevice)
	register(tools.NewConnectionRequest)
	register(tools.NewDeleteTask)
	register(tools.NewRetryTask)

	return s
}

func main() {
	log.Printf("GenieACS MCP %s starting…", version.String())
	// Load config & initialise GenieACS client
	cfg := config.LoadACSConfig()
	acs := client.NewACS(cfg.BaseURL, cfg.User, cfg.Pass)
	s := newMCPServer(acs, cfg.DeviceLimit)

	transport := strings.ToLower(os.Getenv("TRANSPORT"))
	if transport == "stdio" {
		stdioSrv := server.NewStdioServer(s)
		log.Println("GenieACS MCP bridge running on stdio")
		if err := stdioSrv.Listen(context.Background(), os.Stdin, os.Stdout); err != nil {
			log.Fatalf("stdio server error: %v", err)
		}
		return
	}

	httpSrv := server.NewStreamableHTTPServer(s)
	env := loadHTTPEnv()
	addr, srv, err := newHTTPServer(httpSrv, env)
	if err != nil {
		log.Fatalf("%v", err)
	}

	authState := "no auth token"
	if env.authToken != "" {
		authState = "auth enabled"
	}

	log.Printf("GenieACS MCP bridge listening on %s/mcp (Host/Origin guard enabled, %s)", addr, authState)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
