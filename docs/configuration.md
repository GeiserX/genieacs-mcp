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

## Client configuration

Over stdio, which the npm package always uses, a client that reads an `mcpServers` block starts the
server itself:

```json
{
  "mcpServers": {
    "genieacs": {
      "command": "npx",
      "args": ["-y", "genieacs-mcp"],
      "env": {
        "ACS_URL": "http://localhost:7557",
        "ACS_USER": "admin",
        "ACS_PASS": "admin"
      }
    }
  }
}
```

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
