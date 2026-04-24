# Roadmap

> Tracks planned features, improvements, and ideas for genieacs-mcp.
> Items are roughly ordered by priority within each section. Contributions welcome.

---

## v0.4.0 - File Management & Monitoring

- [ ] **`upload_file` tool** - Upload firmware/config files to GenieACS (PUT /files with binary body and metadata headers)
- [ ] **`delete_file` tool** - Remove files from GenieACS storage (DELETE /files/{name})
- [ ] **`genieacs://files/list` resource** - List all files stored in GenieACS with metadata
- [ ] **`genieacs://config` resource** - Read GenieACS server configuration
- [ ] **`genieacs://device/{id}/summary` resource** - Return only key fields (manufacturer, model, firmware, IP, last inform) without the full device document

## v0.5.0 - Observability & Advanced

- [ ] **Health check endpoint** - `/health` endpoint reporting ACS connectivity status
- [ ] **Structured logging** - JSON-formatted logs with request tracing
- [ ] **Prometheus metrics** - Expose request counts, latencies, and error rates at `/metrics`
- [ ] **Request timeout configuration** - Make HTTP timeout to GenieACS NBI configurable (currently uses Go default)
- [ ] **Pagination support** - Add cursor/offset pagination for device lists and task queries
- [ ] **Virtual parameter support** - Read and manage GenieACS virtual parameters (PUT/DELETE /virtual_parameters/{name})

## Ongoing - Quality & DX

- [ ] **Integration tests** - Test tool and resource handlers against a mock ACS server
- [ ] **Resource handler tests** - Unit tests for all seven resource handlers (currently untested)
- [ ] **Tool handler tests** - Unit tests for all twelve tool handlers (currently untested)
- [ ] **golangci-lint in CI** - Add linter beyond the current `go vet` / `go test`
- [ ] **MCP protocol conformance tests** - Validate JSON-RPC responses against the MCP spec
- [ ] **Docker Compose dev environment** - Spin up GenieACS + simulator + MCP server for local development
- [ ] **Streamable HTTP session management** - Proper session lifecycle with cleanup for HTTP transport

## Ideas / Backlog

- [ ] **Device diff tool** - Compare parameter trees between two devices
- [ ] **Bulk reboot tool** - Reboot multiple devices matching a filter in a single call
- [ ] **Audit log resource** - Expose who changed what and when via MCP
- [ ] **mTLS support** - Client certificate auth for the NBI connection
- [ ] **Multi-ACS support** - Connect to multiple GenieACS instances from a single MCP server
- [ ] **SSE transport** - Server-Sent Events transport as an alternative to stdio and HTTP
- [ ] **OpenTelemetry tracing** - Distributed tracing across MCP client -> MCP server -> ACS

## Done

- [x] Core MCP server with stdio and HTTP transports
- [x] `reboot_device`, `download_firmware`, `refresh_parameter` tools
- [x] `genieacs://device/{id}`, `genieacs://file/{name}`, `genieacs://tasks/{id}`, `genieacs://devices/list` resources
- [x] npm distribution with platform-specific Go binary download
- [x] Docker multi-arch image (linux/amd64, linux/arm64)
- [x] CI with tests and codecov
- [x] GoReleaser-based release pipeline
- [x] Published to Official MCP Registry, Glama, MCPServers.org, mcp.so, ToolSDK, awesome-mcp-servers
- [x] Rich TDQS-compliant tool and resource descriptions (purpose, guidelines, limitations, examples)
- [x] Dynamic MCP server version from build metadata (no more hardcoded version string)
- [x] `set_parameter`, `get_parameter` tools — full TR-069 parameter read/write
- [x] `manage_preset`, `manage_provision` tools — preset and provision CRUD
- [x] `genieacs://presets/list`, `genieacs://provisions/list`, `genieacs://faults/{id}` resources
- [x] `search_devices` tool — MongoDB-style device queries
- [x] `tag_device` tool — add/remove device tags
- [x] `connection_request` tool — wake CPE without waiting for periodic inform
- [x] `delete_task`, `retry_task` tools — task lifecycle management
- [x] Configurable device list limit via `DEVICE_LIMIT` environment variable
