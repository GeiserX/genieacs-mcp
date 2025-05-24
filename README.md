# genieacs-mcp
MCP Server for GenieACS in Go

To configure with Clients:

```json
{
  "schema_version": "v1",
  "name_for_human": "GenieACS-MCP",
  "name_for_model": "genieacs_mcp",
  "description_for_human": "Read data from GenieACS and run actions on CPEs (reboot, firmware update, parameter refresh).",
  "description_for_model": "Interact with an Auto-Configuration-Server (ACS) that manages routers. First call initialize, then reuse the returned session id in header \"Mcp-Session-Id\" for every other call. Use readResource to fetch URIs that begin with genieacs://. Use listTools to discover available actions and callTool to execute them.",
  "auth": { "type": "none" },
  "api": {
    "type": "jsonrpc-mcp",
    "url":  "http://localhost:8080/mcp",
    "protocol": "2.0",
    "init_method": "initialize",
    "session_header": "Mcp-Session-Id"
  },
  "logo_url": "https://raw.githubusercontent.com/geiserx/genieacs-docker/main/extra/logo.png",
  "contact_email": "acsdesk@protonmail.com",
  "legal_info_url": "https://github.com/GeiserX/genieacs-mcp/blob/main/LICENSE"
}
```