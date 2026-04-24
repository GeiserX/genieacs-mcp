package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/geiserx/genieacs-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// RegisterFile wires the /file/{name} resource into the server.
func RegisterFile(s *server.MCPServer, acs *client.ACSClient) {
	tpl := mcp.NewResourceTemplate(
		"genieacs://file/{name}",
		"Firmware / file metadata",
		mcp.WithTemplateDescription(
			"Returns JSON metadata for a firmware image or configuration file stored in GenieACS. "+
				"Includes the filename, file type, OUI, product class, version, and upload timestamp. "+
				"Use this resource to verify a file exists before pushing it to a device with the "+
				"download_firmware tool. The {name} parameter is the filename as stored in GenieACS "+
				"(e.g. \"firmware-v2.0.bin\"). Returns a 404 error if the file does not exist.",
		),
		mcp.WithTemplateMIMEType("application/json"),
	)

	s.AddResourceTemplate(tpl, func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
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
}
