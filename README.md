<p align="center">
  <img src="https://raw.githubusercontent.com/GeiserX/genieacs-mcp/main/docs/images/banner.svg" alt="genieacs-mcp" width="100%">
</p>

<p align="center">
  <a href="https://www.npmjs.com/package/genieacs-mcp"><img src="https://img.shields.io/npm/v/genieacs-mcp?style=flat-square&logo=npm" alt="npm"></a>
  <a href="https://github.com/GeiserX/genieacs-mcp/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/GeiserX/genieacs-mcp/ci.yml?style=flat-square&logo=github&label=CI" alt="CI"></a>
  <a href="https://codecov.io/gh/GeiserX/genieacs-mcp"><img src="https://img.shields.io/codecov/c/github/GeiserX/genieacs-mcp?style=flat-square&logo=codecov&label=Coverage" alt="Coverage"></a>
  <a href="https://pkg.go.dev/github.com/geiserx/genieacs-mcp"><img src="https://pkg.go.dev/badge/github.com/geiserx/genieacs-mcp.svg" alt="Go Reference"></a>
  <a href="https://github.com/GeiserX/genieacs-mcp/blob/main/LICENSE"><img src="https://img.shields.io/github/license/GeiserX/genieacs-mcp?style=flat-square" alt="License"></a>
  <a href="https://hub.docker.com/r/drumsergio/genieacs-mcp"><img src="https://img.shields.io/docker/pulls/drumsergio/genieacs-mcp?style=flat-square&logo=docker" alt="Docker Pulls"></a>
  <a href="https://github.com/GeiserX/genieacs-mcp/stargazers"><img src="https://img.shields.io/github/stars/GeiserX/genieacs-mcp?style=flat-square&logo=github" alt="GitHub Stars"></a>
</p>

**genieacs-mcp** is an MCP server for [GenieACS](https://github.com/genieacs/genieacs), the open-source TR-069 ACS that manages routers, ONTs and other CPE devices. Add it to Claude Desktop, Cursor or any other MCP client and ask in plain words: which devices stopped informing last night, what firmware serial 000003 runs, reboot the ones tagged `pilot`. The assistant reads and acts through your GenieACS NBI with 7 resources and 12 tools; one Go binary, no database of its own.

<p align="center">
  <img src="https://raw.githubusercontent.com/GeiserX/genieacs-mcp/main/docs/images/screenshots/search-devices.png" alt="MCP Inspector connected to genieacs-mcp: the search_devices tool run with the query _tags residential, and the result listing simulated Huawei BM632w devices with their ids, manufacturer, last inform and tags" width="100%">
</p>

## Features

- Find devices the way you would ask a colleague: by tag, model, firmware, last inform, or any TR-069 parameter (`search_devices` takes a MongoDB-style filter).
- Read a device's full parameter tree, its tasks and its faults, plus the presets, provisions and files on the ACS, as resources (`genieacs://device/{id}`, `genieacs://devices/list`, ...).
- Reboot a device, push firmware to it, refresh or set a parameter, or wake it with a connection request, from a chat.
- Tag devices, create or delete presets and provisions, and delete or retry tasks, so the assistant can fix what it finds.
- Every tool description says what it does, its limits and an example, so clients pick the right one without a prompt from you.
- Read versus act: `search_devices`, `get_parameter` and the seven resources only read; the other ten tools queue a task on a device or change the ACS, and each tool's hints say which, so your client can ask before the ones that act.
- Read-only mode: `GENIEACS_READ_ONLY=true` serves only the two reading tools and the resources; `GENIEACS_TOOLS=search_devices,reboot_device` serves only the tools you name.
- Speaks MCP 2026-07-28 and every earlier version back to 2024-11-05, so new and old clients both connect, over stdio or stateless HTTP.
- Safe on a laptop: the HTTP transport listens on loopback only unless you set an address and a bearer token, and checks `Host` and `Origin` against DNS rebinding.
- One static Go binary for Linux, macOS and Windows on amd64 and arm64, also as an npm package (`npx genieacs-mcp`) and a Docker image.

## Quick start

You need a GenieACS whose NBI (port 7557) this machine can reach; without one, the family's demo stack gives you an ACS with a simulated router in about a minute:

```sh
curl -fsSLO https://raw.githubusercontent.com/GeiserX/genieacs-container/main/docker-compose.yml
docker compose --profile testing up -d      # GenieACS + MongoDB + one simulated CPE, NBI on localhost:7557
```

Then add the server to a client that reads an `mcpServers` block (Claude Desktop, Cursor, Claude Code; in VS Code the `.vscode/mcp.json` key is `servers`):

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

The same server in other clients:

```sh
# Claude Code
claude mcp add --transport stdio --env ACS_URL=http://localhost:7557 genieacs -- npx -y genieacs-mcp

# Docker, over stdio (any client that starts a command)
docker run -i --rm -e ACS_URL=http://host.docker.internal:7557 drumsergio/genieacs-mcp:v0.4.0
```

```jsonc
// VS Code, .vscode/mcp.json
{ "servers": { "genieacs": { "type": "stdio", "command": "npx", "args": ["-y", "genieacs-mcp"], "env": { "ACS_URL": "http://localhost:7557" } } } }
```

```toml
# Codex, ~/.codex/config.toml
[mcp_servers.genieacs]
command = "npx"
args = ["-y", "genieacs-mcp"]
env = { ACS_URL = "http://localhost:7557" }
```

It worked when you ask "which devices does the ACS know about?" and the assistant answers with device ids such as `202BC1-BM632w-000000` (the demo stack) or your own. Docker, the HTTP transport and a local build are in [Getting started](https://geiserx.github.io/genieacs-mcp/getting-started/).

## Documentation

The full documentation is at [geiserx.github.io/genieacs-mcp](https://geiserx.github.io/genieacs-mcp/).

- [Getting started](https://geiserx.github.io/genieacs-mcp/getting-started/): npm, Docker Compose, local build, and what the first exchange looks like
- [Configuration](https://geiserx.github.io/genieacs-mcp/configuration/): every environment variable, the HTTP transport's security model, client config for stdio and HTTP
- [Usage](https://geiserx.github.io/genieacs-mcp/usage/): the 7 resources and 12 tools with their arguments, which ones act on devices, example prompts
- [Development](https://geiserx.github.io/genieacs-mcp/development/): tests, trying the server with MCP Inspector, contributing
- [Related projects](https://geiserx.github.io/genieacs-mcp/related/): the GenieACS family, other MCP servers, where this server is listed

## Related projects

Listed in the [official MCP Registry](https://registry.modelcontextprotocol.io) as `io.github.GeiserX/genieacs-mcp`. Part of the GenieACS family: [genieacs-container](https://github.com/GeiserX/genieacs-container) (the ACS itself, Docker and Helm, with this server as a compose profile), [genieacs-sim-container](https://github.com/GeiserX/genieacs-sim-container), [genieacs-ansible](https://github.com/GeiserX/genieacs-ansible), [genieacs-ha](https://github.com/GeiserX/genieacs-ha), [genieacs-services](https://github.com/GeiserX/genieacs-services), and [n8n-nodes-genieacs](https://github.com/GeiserX/n8n-nodes-genieacs) (archived).

## License

[GPL-3.0-or-later](https://github.com/GeiserX/genieacs-mcp/blob/main/LICENSE)
