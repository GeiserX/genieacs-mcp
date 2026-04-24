# Roadmap

> Tracks planned features, improvements, and ideas for genieacs-mcp.
> Items are roughly ordered by priority within each section. Contributions welcome.

---

## v0.2.0 - Parameter Management & Presets

- [ ] **`set_parameter` tool** - Set one or more TR-069 parameter values on a device (setParameterValues task)
- [ ] **`get_parameter` tool** - Read specific parameter values from a device without refreshing from CPE (read from ACS cache)
- [ ] **`manage_preset` tool** - Create, update, and delete presets (filter + provision mappings)
- [ ] **`manage_provision` tool** - Upload and manage provision scripts
- [ ] **`genieacs://presets/list` resource** - List all configured presets with their preconditions and provisions
- [ ] **`genieacs://provisions/list` resource** - List all provision scripts stored in GenieACS
- [ ] **`genieacs://faults/{id}` resource** - Retrieve fault records for a device to diagnose failed tasks

## v0.3.0 - Bulk Operations & Filtering

- [ ] **`genieacs://devices/search` resource template** - Query devices with MongoDB-style filters (e.g. by tag, manufacturer, model)
- [ ] **`bulk_reboot` tool** - Reboot multiple devices matching a filter in a single call
- [ ] **`tag_device` tool** - Add or remove tags on a device
- [ ] **Configurable device list limit** - Make the 500-device hard limit in `genieacs://devices/list` configurable via environment variable
- [ ] **Pagination support** - Add cursor/offset pagination for device lists and task queries

## v0.4.0 - File Management & Monitoring

- [ ] **`upload_file` tool** - Upload firmware/config files to GenieACS (currently only download/push is supported)
- [ ] **`delete_file` tool** - Remove files from GenieACS storage
- [ ] **`genieacs://files/list` resource** - List all files stored in GenieACS with metadata
- [ ] **`genieacs://config` resource** - Read GenieACS server configuration
- [ ] **`genieacs://device/{id}/summary` resource** - Return only key fields (manufacturer, model, firmware, IP, last inform) without the full device document

## v0.5.0 - Connection & Observability

- [ ] **Connection request tool** - Trigger an immediate connection request to wake a CPE without waiting for periodic inform
- [ ] **Health check endpoint** - `/health` endpoint reporting ACS connectivity status
- [ ] **Structured logging** - JSON-formatted logs with request tracing
- [ ] **Prometheus metrics** - Expose request counts, latencies, and error rates at `/metrics`
- [ ] **Request timeout configuration** - Make HTTP timeout to GenieACS NBI configurable (currently uses Go default)

## Ongoing - Quality & DX

- [ ] **Integration tests** - Test tool and resource handlers against a mock ACS server
- [ ] **Resource handler tests** - Unit tests for all four resource handlers (currently untested)
- [ ] **Tool handler tests** - Unit tests for all three tool handlers (currently untested)
- [ ] **golangci-lint in CI** - Add linter beyond the current `go vet` / `go test`
- [ ] **MCP protocol conformance tests** - Validate JSON-RPC responses against the MCP spec
- [ ] **Docker Compose dev environment** - Spin up GenieACS + simulator + MCP server for local development
- [ ] **Streamable HTTP session management** - Proper session lifecycle with cleanup for HTTP transport

## Ideas / Backlog

- [ ] **Virtual parameter support** - Read and manage GenieACS virtual parameters
- [ ] **Device diff tool** - Compare parameter trees between two devices
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
