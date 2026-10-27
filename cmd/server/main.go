// Command server runs the GenieACS MCP server. It serves the MCP protocol over
// streamable HTTP on 127.0.0.1:8080 by default, or over stdin and stdout when
// TRANSPORT=stdio, and reads its GenieACS settings from the environment.
package main

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

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
	// An empty or wildcard host (e.g. ":8080", "0.0.0.0:8080", "[::]:8080")
	// binds all interfaces, so it must NOT be treated as loopback — otherwise
	// the listener would start LAN-wide without requiring MCP_AUTH_TOKEN.
	if isWildcardHost(host) {
		return false
	}
	if host == "localhost" {
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

// maxRequestBody caps one HTTP request body. The SDK reads the whole body
// into memory, so without a cap one large POST could exhaust it.
const maxRequestBody = 4 << 20

// buildHTTPHandler wraps the MCP handler for the HTTP transport, outermost
// first: a DNS-rebinding guard (Host/Origin validation derived from addr plus
// the MCP_ALLOWED_HOSTS / MCP_ALLOWED_ORIGINS allowlists), the standard
// library's cross-origin protection (which also rejects cross-site browser
// requests that carry Sec-Fetch-Site but no Origin), the bearer-auth check
// when authToken is set, and a request body cap.
func buildHTTPHandler(mcpHandler http.Handler, addr, authToken, extraHosts, extraOrigins string) (http.Handler, error) {
	allowedHosts := allowedHostSet(addr, splitList(extraHosts))
	allowedOrigins := allowedOriginSet(allowedHosts, splitList(extraOrigins))

	cop := http.NewCrossOriginProtection()
	for origin := range allowedOrigins {
		if err := cop.AddTrustedOrigin(origin); err != nil {
			return nil, fmt.Errorf("MCP_ALLOWED_ORIGINS: %w", err)
		}
	}

	handler := http.MaxBytesHandler(mcpHandler, maxRequestBody)
	if authToken != "" {
		handler = bearerAuth(handler, authToken)
	}
	return dnsRebindGuard(cop.Handler(handler), allowedHosts, allowedOrigins), nil
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
	addr := config.Getenv("MCP_LISTEN_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8080"
	}
	return httpEnv{
		addr:          addr,
		authToken:     config.Getenv("MCP_AUTH_TOKEN"),
		allowedHosts:  config.Getenv("MCP_ALLOWED_HOSTS"),
		allowedOrigin: config.Getenv("MCP_ALLOWED_ORIGINS"),
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
	handler, err := buildHTTPHandler(mcpHandler, env.addr, env.authToken, env.allowedHosts, env.allowedOrigin)
	if err != nil {
		return "", nil, err
	}

	mux := http.NewServeMux()
	mux.Handle("/mcp", handler)
	return env.addr, &http.Server{
		Addr:    env.addr,
		Handler: mux,
		// Bound how long a client may hold a connection without sending a
		// full request. No WriteTimeout: responses may stream.
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}, nil
}

var errAuthTokenRequired = errors.New("MCP_AUTH_TOKEN is required when MCP_LISTEN_ADDR is not loopback")

// bearerAuth requires "Authorization: Bearer <token>". The scheme is matched
// case-insensitively (RFC 9110), a token sent without it is refused, and the
// token is compared in constant time.
func bearerAuth(next http.Handler, token string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scheme, got, ok := strings.Cut(r.Header.Get("Authorization"), " ")
		if !ok || !strings.EqualFold(scheme, "Bearer") ||
			subtle.ConstantTimeCompare([]byte(strings.TrimSpace(got)), []byte(token)) != 1 {
			w.Header().Set("WWW-Authenticate", `Bearer realm="genieacs-mcp"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// serverOptions are the settings newMCPServer takes from the environment.
type serverOptions struct {
	deviceLimit int
}

// newMCPServer builds the MCP server with all GenieACS resources and tools
// registered against acs.
func newMCPServer(acs *client.ACSClient, opts serverOptions) (*server.MCPServer, error) {
	deviceLimit := opts.deviceLimit
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

	return s, nil
}

// newStreamableHandler serves s over Streamable HTTP. It is stateless: the
// server keeps nothing per client, so no session ids are minted or stored,
// and 2026-07-28 requests and legacy initialize clients are served alike.
func newStreamableHandler(s *server.MCPServer) http.Handler {
	return server.NewStreamableHTTPServer(s, server.WithStateLess(true))
}

// runStdio serves the MCP server over the stdio transport until ctx is
// cancelled or the input stream closes.
func runStdio(ctx context.Context, s *server.MCPServer, in io.Reader, out io.Writer) error {
	log.Println("GenieACS MCP bridge running on stdio")
	return server.NewStdioServer(s).Listen(ctx, in, out)
}

// serveHTTP starts srv and blocks until ctx is cancelled, then shuts the
// server down gracefully. A clean shutdown (http.ErrServerClosed) is not
// treated as an error.
func serveHTTP(ctx context.Context, srv *http.Server) error {
	errCh := make(chan error, 1)
	go func() {
		err := srv.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		errCh <- err
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

// run wires the MCP server to the requested transport. It is the testable
// core of main(): all configuration comes from the environment, but the
// blocking serve loops honour ctx so callers (and tests) can stop them.
func run(ctx context.Context, s *server.MCPServer, in io.Reader, out io.Writer) error {
	if strings.ToLower(config.Getenv("TRANSPORT")) == "stdio" {
		return runStdio(ctx, s, in, out)
	}

	env := loadHTTPEnv()
	addr, srv, err := newHTTPServer(newStreamableHandler(s), env)
	if err != nil {
		return err
	}

	authState := "no auth token"
	if env.authToken != "" {
		authState = "auth enabled"
	}
	log.Printf("GenieACS MCP bridge listening on %s/mcp (Host/Origin guard enabled, %s)", addr, authState)
	return serveHTTP(ctx, srv)
}

func main() {
	log.Printf("GenieACS MCP %s starting…", version.String())
	// Load config & initialise GenieACS client
	cfg := config.LoadACSConfig()
	acs := client.NewACS(cfg.BaseURL, cfg.User, cfg.Pass)
	acs.SetMaxResponseBytes(cfg.MaxResponseBytes)
	acs.SetTimeout(cfg.Timeout)
	s, err := newMCPServer(acs, serverOptions{deviceLimit: cfg.DeviceLimit})
	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}

	// Stop cleanly on SIGINT/SIGTERM so containers shut down gracefully.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, s, os.Stdin, os.Stdout); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
