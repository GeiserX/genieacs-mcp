# Getting started

You need a GenieACS instance whose NBI (port 7557 by default) the server can reach. Pick one way to run
the server; every setting is on [Configuration](configuration.md).

## npm (stdio transport)

```sh
npx -y genieacs-mcp
```

Or install globally:

```sh
npm install -g genieacs-mcp
genieacs-mcp
```

This downloads the pre-built Go binary for your platform and runs it with stdio transport, which any MCP
client can start. The client config block is on [Configuration](configuration.md#client-configuration).

## Docker Compose

The server is the `genieacs-mcp` service, behind the `mcp` profile, in
[genieacs-container](https://github.com/GeiserX/genieacs-container)'s `docker-compose.yml`:

```sh
curl -fsSLO https://raw.githubusercontent.com/GeiserX/genieacs-container/main/docker-compose.yml
docker compose --profile mcp up -d
```

It runs the HTTP transport on port 8080 (`http://localhost:8080/mcp`) and talks to the `genieacs`
service's NBI. The image on its own is `drumsergio/genieacs-mcp:v0.3.3`.

As published, the `mcp` profile listens on loopback inside its container, so `http://localhost:8080/mcp` is
reachable only once the compose file sets `MCP_LISTEN_ADDR: 0.0.0.0:8080` and an `MCP_AUTH_TOKEN`, and from
then on every HTTP client sends that token as `Authorization: Bearer <token>` (see the `headers` block in
[Configuration](configuration.md#client-configuration)) or gets a 401; until that lands in genieacs-container,
use the npm path above, or run the binary on the host with `ACS_URL` pointing at port 7557.

## Local build

```sh
git clone https://github.com/GeiserX/genieacs-mcp
cd genieacs-mcp

# (optional) create .env from the sample
cp .env.example .env && $EDITOR .env

go run ./cmd/server
```

## First run

Add the server to your client, then ask it to list the devices, or read the `genieacs://devices/list`
resource yourself. Over stdio the exchange looks like this (one JSON-RPC message per line):

```text
-> {"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}
<- {"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-06-18","capabilities":{"resources":{},"tools":{"listChanged":true}},"serverInfo":{"name":"GenieACS MCP Bridge",...}}}
-> {"jsonrpc":"2.0","method":"notifications/initialized"}
-> {"jsonrpc":"2.0","id":2,"method":"resources/read","params":{"uri":"genieacs://devices/list"}}
<- {"jsonrpc":"2.0","id":2,"result":{"contents":[{"uri":"genieacs://devices/list","mimeType":"application/json","text":"[{\"_id\":\"202BC1-BM632w-000000\"}]"}]}}
```

When the ACS answers, `text` holds one entry per device, or `[]` when none is registered yet. When
`ACS_URL` is wrong, `initialize` still succeeds and the read fails with the connection error, for example:

```text
<- {"jsonrpc":"2.0","id":2,"error":{"code":-32603,"message":"Get \"http://nothere:7557/devices?...\": dial tcp: lookup nothere ...: no such host"}}
```

The server also logs a warning at startup when `ACS_USER` and `ACS_PASS` are not set.
