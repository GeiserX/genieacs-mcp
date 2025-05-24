package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/geiserx/genieacs-mcp/client"
	"github.com/geiserx/genieacs-mcp/config"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
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

	// Declare a *dynamic* resource template:
	// URI scheme: genieacs://device/{id}
	deviceTpl := mcp.NewResourceTemplate(
		"genieacs://device/{id}",
		"GenieACS device JSON",
		mcp.WithTemplateDescription("Raw device document as returned by NBI"),
		mcp.WithTemplateMIMEType("application/json"),
	)

	// Attach handler
	s.AddResourceTemplate(deviceTpl,
		func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			id := strings.TrimPrefix(req.Params.URI, "genieacs://device/")
			if id == "" {
				return nil, fmt.Errorf("missing device id")
			}

			body, err := acs.GetDevice(id)
			if err != nil {
				return nil, err
			}

			return []mcp.ResourceContents{
				mcp.TextResourceContents{
					URI:      fmt.Sprintf("genieacs://device/%s", id),
					MIMEType: "application/json",
					Text:     string(body),
				},
			}, nil
		})

	//----------------------------------------------------
	// File metadata  –  genieacs://file/{name}
	//----------------------------------------------------
	fileTpl := mcp.NewResourceTemplate(
		"genieacs://file/{name}",
		"Firmware / file metadata",
		mcp.WithTemplateDescription("Metadata for a file stored in GenieACS"),
		mcp.WithTemplateMIMEType("application/json"),
	)

	s.AddResourceTemplate(fileTpl, func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		name := strings.TrimPrefix(req.Params.URI, "genieacs://file/")
		if name == "" {
			return nil, fmt.Errorf("missing file name")
		}

		body, err := acs.GetFileByName(name)
		if err != nil {
			return nil, err
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     string(body),
			},
		}, nil
	})

	//----------------------------------------------------
	// Device tasks  –  genieacs://tasks/{deviceId}
	//----------------------------------------------------
	taskTpl := mcp.NewResourceTemplate(
		"genieacs://tasks/{id}",
		"Pending and historical tasks for a device",
		mcp.WithTemplateMIMEType("application/json"),
	)

	s.AddResourceTemplate(taskTpl, func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		id := strings.TrimPrefix(req.Params.URI, "genieacs://tasks/")
		if id == "" {
			return nil, fmt.Errorf("missing device id")
		}

		body, err := acs.GetTasksForDevice(id)
		if err != nil {
			return nil, err
		}

		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     string(body),
			},
		}, nil
	})

	//----------------------------------------------------
	// Device catalogue  –  genieacs://devices/list
	//----------------------------------------------------
	devicesRes := mcp.NewResource(
		"genieacs://devices/list",
		"Device summary list",
		mcp.WithResourceDescription("Lightweight list of devices and last inform time"),
		mcp.WithMIMEType("application/json"),
	)

	s.AddResource(devicesRes, func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		body, err := acs.ListDeviceSummaries(500) // hard-coded limit to 500
		if err != nil {
			return nil, err
		}
		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      "genieacs://devices/list",
				MIMEType: "application/json",
				Text:     string(body),
			},
		}, nil
	})

	// -----------------------------------------------------------------
	// TOOL: reboot_device
	// -----------------------------------------------------------------
	rebootTool := mcp.NewTool("reboot_device",
		mcp.WithDescription("Reboot a CPE via GenieACS"),
		mcp.WithString("device_id",
			mcp.Required(),
			mcp.Description("Exact device ID (_id) as known by GenieACS"),
		),
	)

	s.AddTool(rebootTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		devID, err := req.RequireString("device_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		resp, err := acs.RebootDevice(devID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Reboot task queued/processed. Raw ACS response: %s", string(resp))), nil
	})

	// -----------------------------------------------------------------
	// TOOL: download_firmware
	// -----------------------------------------------------------------
	dlTool := mcp.NewTool("download_firmware",
		mcp.WithDescription("Push a firmware (or any file) to a device"),
		mcp.WithString("device_id",
			mcp.Required(),
			mcp.Description("Target device ID"),
		),
		mcp.WithString("file_id",
			mcp.Required(),
			mcp.Description("GridFS _id of the file to download"),
		),
		mcp.WithString("filename",
			mcp.Description("(optional) filename to pass to the CPE"),
		),
	)

	s.AddTool(dlTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		devID, err := req.RequireString("device_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		fileID, err := req.RequireString("file_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		name := ""
		if v, _ := req.GetArguments()["filename"].(string); v != "" {
			name = v
		}

		resp, err := acs.DownloadFirmware(devID, fileID, name)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Download task queued/processed. Raw ACS response: %s", string(resp))), nil
	})

	// -----------------------------------------------------------------
	// TOOL: refresh_parameter
	// -----------------------------------------------------------------
	refreshTool := mcp.NewTool("refresh_parameter",
		mcp.WithDescription("Ask the CPE to send a fresh copy of a single parameter"),
		mcp.WithString("device_id",
			mcp.Required(),
			mcp.Description("Target device ID"),
		),
		mcp.WithString("parameter",
			mcp.Required(),
			mcp.Description("Full TR-069 path to refresh, e.g. Device.DeviceInfo.SerialNumber"),
		),
	)

	s.AddTool(refreshTool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		devID, err := req.RequireString("device_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		param, err := req.RequireString("parameter")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		resp, err := acs.RefreshParameter(devID, param)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Refresh task queued/processed. Raw ACS response: %s", string(resp))), nil
	})

	httpSrv := server.NewStreamableHTTPServer(s) // optional: WithHTTPContextFunc(...)

	log.Println("GenieACS MCP bridge listening on :8080")
	if err := httpSrv.Start(":8080"); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
