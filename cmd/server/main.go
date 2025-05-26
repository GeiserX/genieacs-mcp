package main

import (
	"log"

	"github.com/geiserx/genieacs-mcp/client"
	"github.com/geiserx/genieacs-mcp/config"
	"github.com/geiserx/genieacs-mcp/internal/resources"
	"github.com/geiserx/genieacs-mcp/internal/tools"
	"github.com/geiserx/genieacs-mcp/version"
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
		"0.0.1",
		server.WithToolCapabilities(true),
		server.WithRecovery(),
	)

	//----------------------------------------------------
	// Register device: genieacs://device/{id}
	//----------------------------------------------------
	resources.RegisterDevice(s, acs)

	//----------------------------------------------------
	// File metadata  –  genieacs://file/{name}
	//----------------------------------------------------
	resources.RegisterFile(s, acs)

	//----------------------------------------------------
	// Device tasks  –  genieacs://tasks/{deviceId}
	//----------------------------------------------------
	resources.RegisterTasks(s, acs)

	//----------------------------------------------------
	// Device catalogue  –  genieacs://devices/list
	//----------------------------------------------------
	resources.RegisterCatalogue(s, acs)

	// -----------------------------------------------------------------
	// TOOL: reboot_device
	// -----------------------------------------------------------------
	tool, handler := tools.NewReboot(acs)
	s.AddTool(tool, handler)

	// -----------------------------------------------------------------
	// TOOL: download_firmware
	// -----------------------------------------------------------------
	tool, handler = tools.NewDownloadFirmware(acs) // <— new lines
	s.AddTool(tool, handler)

	// -----------------------------------------------------------------
	// TOOL: refresh_parameter
	// -----------------------------------------------------------------
	tool, handler = tools.NewRefreshParameter(acs)
	s.AddTool(tool, handler)

	httpSrv := server.NewStreamableHTTPServer(s)
	log.Println("GenieACS MCP bridge listening on :8080")
	if err := httpSrv.Start(":8080"); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
