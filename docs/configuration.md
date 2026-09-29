# Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `ACS_URL` | http://localhost:7557 | GenieACS NBI endpoint (without trailing /) |
| `ACS_USER` | _(empty)_ | GenieACS NBI basic-auth username |
| `ACS_PASS` | _(empty)_ | GenieACS NBI basic-auth password |
| `TRANSPORT` | _(empty = HTTP)_ | Set to `stdio` for stdio transport |
| `DEVICE_LIMIT` | 500 | Max devices returned by `genieacs://devices/list` |
| `MCP_LISTEN_ADDR` | 127.0.0.1:8080 | HTTP listen address (only used when TRANSPORT is not stdio) |
| `MCP_AUTH_TOKEN` | _(empty)_ | Bearer token for HTTP transport auth. **Required** when `MCP_LISTEN_ADDR` is non-loopback |
| `MCP_ALLOWED_HOSTS` | _(empty)_ | Comma-separated extra `Host` header values to accept (e.g. a reverse-proxy domain). Loopback names on the listen port are always allowed |
| `MCP_ALLOWED_ORIGINS` | _(empty)_ | Comma-separated extra browser `Origin` values to accept (e.g. `https://my-ai-app.com`) |

> **Security — HTTP transport.** The HTTP transport validates the `Host` and
> `Origin` headers on every request to prevent [DNS rebinding](https://en.wikipedia.org/wiki/DNS_rebinding)
> from a malicious web page reaching a local listener. Requests with an
> untrusted `Host`, or a present-but-untrusted `Origin`, are rejected with
> `403`. Loopback access works with no configuration; if you expose the server
> through a reverse proxy or a hostname, add that name to `MCP_ALLOWED_HOSTS`
> (and `MCP_ALLOWED_ORIGINS` for browser clients). The `stdio` transport is
> unaffected and remains the recommended mode for local MCP clients.

Put them in a `.env` file (from `.env.example`) or set them in the environment. 

## Example configuration for client LLMs

```json
{
  "schema_version": "v1",
  "name_for_human": "GenieACS-MCP",
  "name_for_model": "genieacs_mcp",
  "description_for_human": "Full CPE management through GenieACS — parameter read/write, presets, provisions, firmware, tags, search, and task lifecycle.",
  "description_for_model": "Interact with a GenieACS TR-069 Auto-Configuration-Server (ACS) that manages CPE devices (routers, ONTs, gateways). First call initialize, then reuse the returned session id in header \"Mcp-Session-Id\" for every other call. Use readResource to fetch URIs that begin with genieacs:// (devices, presets, provisions, faults). Use listTools to discover available actions (parameter read/write, presets, provisions, tags, search, task management) and callTool to execute them.",
  "auth": { "type": "bearer", "token": "<MCP_AUTH_TOKEN value>" },
  "api": {
    "type": "jsonrpc-mcp",
    "url":  "http://localhost:8080/mcp",
    "init_method": "initialize",
    "session_header": "Mcp-Session-Id"
  },
  "logo_url": "https://raw.githubusercontent.com/GeiserX/genieacs-container/main/extra/logo.png",
  "contact_email": "acsdesk@protonmail.com",
  "legal_info_url": "https://github.com/GeiserX/genieacs-mcp/blob/main/LICENSE"
}
```
