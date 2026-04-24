package main

import (
	"context"
	"log"
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

func main() {
	log.Printf("GenieACS MCP %s starting…", version.String())
	// Load config & initialise GenieACS client
	cfg := config.LoadACSConfig()
	acs := client.NewACS(cfg.BaseURL, cfg.User, cfg.Pass)
	// Create MCP server
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
	resources.RegisterCatalogue(s, acs, cfg.DeviceLimit)
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

	transport := strings.ToLower(os.Getenv("TRANSPORT"))
	if transport == "stdio" {
		stdioSrv := server.NewStdioServer(s)
		log.Println("GenieACS MCP bridge running on stdio")
		if err := stdioSrv.Listen(context.Background(), os.Stdin, os.Stdout); err != nil {
			log.Fatalf("stdio server error: %v", err)
		}
	} else {
		httpSrv := server.NewStreamableHTTPServer(s)
		addr := os.Getenv("MCP_LISTEN_ADDR")
		if addr == "" {
			addr = "127.0.0.1:8080"
		}
		log.Printf("GenieACS MCP bridge listening on %s", addr)
		if err := httpSrv.Start(addr); err != nil {
			log.Fatalf("server error: %v", err)
		}
	}
}
