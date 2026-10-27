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
| `MCP_ALLOWED_ORIGINS` | _(empty)_ | Comma-separated extra browser `Origin` values to accept (e.g. `https://my-ai-app.com`). A value that is not a plain `scheme://host[:port]` origin stops the server at startup |
| `GENIEACS_READ_ONLY` | false | `true` (or `1`, `yes`, `on`) serves only the tools that never change a device or the ACS: `search_devices` and `get_parameter`. The resources are always served |
| `GENIEACS_TOOLS` | _(empty = all)_ | Comma-separated tool names to serve, e.g. `search_devices,get_parameter,reboot_device`. An unknown name stops the server at startup and the error lists the known ones. With `GENIEACS_READ_ONLY` a tool must pass both |
| `ACS_TIMEOUT` | 30 | Seconds to wait for one GenieACS NBI request before the call fails |
| `ACS_MAX_RESPONSE_BYTES` | 4194304 (4 MiB) | Largest NBI response the server reads. A bigger one fails with a message to narrow the query or projection instead of filling memory |

> **Security — HTTP transport.** The HTTP transport validates the `Host` and
> `Origin` headers on every request to prevent [DNS rebinding](https://en.wikipedia.org/wiki/DNS_rebinding)
> from a malicious web page reaching a local listener. Requests with an
> untrusted `Host`, or a present-but-untrusted `Origin`, are rejected with
> `403`. Loopback access works with no configuration; if you expose the server
> through a reverse proxy or a hostname, add that name to `MCP_ALLOWED_HOSTS`
> (and `MCP_ALLOWED_ORIGINS` for browser clients). The `stdio` transport is
> unaffected and remains the recommended mode for local MCP clients.
>
> Every HTTP request also passes Go's cross-origin protection, which turns away a browser request whose
> `Sec-Fetch-Site` says it came from another site. When `MCP_AUTH_TOKEN` is set the request needs the
> header `Authorization: Bearer <token>`; the scheme word is matched case-insensitively and the token in
> constant time, and anything else gets `401` with `WWW-Authenticate: Bearer`. A request body larger than
> 4 MiB is refused.

Ten of the twelve tools act on a device or change the ACS. Each tool says so in its annotations
(`readOnlyHint`, `destructiveHint`, `idempotentHint`), so a client can ask before the ones that act. To
take the acting tools away altogether, set `GENIEACS_READ_ONLY=true`, or list the tools you want in
`GENIEACS_TOOLS`. A tool left out is never registered, so the client never sees it.

The server speaks MCP 2026-07-28 and the earlier versions 2025-11-25, 2025-06-18, 2025-03-26 and
2024-11-05. A new client finds it with `server/discover`, an older one with `initialize`; both work over
stdio and over HTTP. The HTTP transport is stateless and issues no `Mcp-Session-Id`.

Put them in a `.env` file (from `.env.example`) or set them in the environment.

A value that is nothing but an unexpanded placeholder, such as `${ACS_PASS}`, counts as unset. Some MCP clients
write that placeholder when an optional field is left blank; the server ignores it rather than send it to GenieACS
as a password or accept it as the HTTP auth token.

## Client configuration

Over stdio, which the npm package always uses, a client that reads an `mcpServers` block starts the
server itself:

```json
{
  "mcpServers": {
    "genieacs": {
      "command": "npx",
      "args": ["-y", "genieacs-mcp"],
      "env": { "ACS_URL": "http://localhost:7557" }
    }
  }
}
```

Add `ACS_USER` and `ACS_PASS` only if you put basic auth in front of the NBI; GenieACS itself ships the NBI
without authentication, and the `admin` / `admin` login of the web UI is not an NBI credential.

Over HTTP, for a server you already run (the Docker image, or the binary without `TRANSPORT=stdio`), a
client that supports remote servers connects to the `/mcp` endpoint. Drop `headers` if you did not set
`MCP_AUTH_TOKEN`:

```json
{
  "mcpServers": {
    "genieacs": {
      "url": "http://localhost:8080/mcp",
      "headers": { "Authorization": "Bearer <MCP_AUTH_TOKEN>" }
    }
  }
}
```
