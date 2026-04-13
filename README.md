<p align="center">
  <img src="docs/images/banner.svg" alt="GenieACS MCP banner" width="900"/>
</p>

<h1 align="center">GenieACS-MCP</h1>

<p align="center">
  <a href="https://www.npmjs.com/package/genieacs-mcp"><img src="https://img.shields.io/npm/v/genieacs-mcp?style=flat-square&logo=npm" alt="npm"/></a>
  <a href="https://github.com/GeiserX/genieacs-mcp/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/GeiserX/genieacs-mcp/ci.yml?style=flat-square&logo=github&label=CI" alt="CI"/></a>
  <a href="https://codecov.io/gh/GeiserX/genieacs-mcp"><img src="https://img.shields.io/codecov/c/github/GeiserX/genieacs-mcp?style=flat-square&logo=codecov&label=Coverage" alt="Coverage"/></a>
  <img src="https://img.shields.io/badge/Go-1.24-blue?style=flat-square&logo=go&logoColor=white" alt="Go"/>
  <a href="https://hub.docker.com/r/drumsergio/genieacs-mcp"><img src="https://img.shields.io/docker/pulls/drumsergio/genieacs-mcp?style=flat-square&logo=docker" alt="Docker Pulls"/></a>
  <a href="https://github.com/GeiserX/genieacs-mcp/stargazers"><img src="https://img.shields.io/github/stars/GeiserX/genieacs-mcp?style=flat-square&logo=github" alt="GitHub Stars"/></a>
  <a href="https://github.com/GeiserX/genieacs-mcp/blob/main/LICENSE"><img src="https://img.shields.io/github/license/GeiserX/genieacs-mcp?style=flat-square" alt="License"/></a>
</p>
<p align="center">
  <a href="https://registry.modelcontextprotocol.io"><img src="https://img.shields.io/badge/MCP-Official%20Registry-E6522C?style=flat-square" alt="Official MCP Registry"/></a>
  <a href="https://glama.ai/mcp/servers/GeiserX/genieacs-mcp"><img src="https://glama.ai/mcp/servers/GeiserX/genieacs-mcp/badges/score.svg" alt="Glama MCP Server" /></a>
  <a href="https://mcpservers.org/servers/geiserx/genieacs-mcp"><img src="https://img.shields.io/badge/MCPServers.org-listed-green?style=flat-square" alt="MCPServers.org"/></a>
  <a href="https://mcp.so/server/genieacs-mcp"><img src="https://img.shields.io/badge/mcp.so-listed-blue?style=flat-square" alt="mcp.so"/></a>
  <a href="https://github.com/toolsdk-ai/toolsdk-mcp-registry"><img src="https://img.shields.io/badge/ToolSDK-Registry-orange?style=flat-square" alt="ToolSDK Registry"/></a>
</p>

<p align="center"><strong>A tiny bridge that exposes any GenieACS instance as an MCP v1 (JSON-RPC for LLMs) server written in Go.</strong></p>

---

## ✨ What you get

| Type            | What for                                                                   | MCP URI / Tool id                |
|-----------------|----------------------------------------------------------------------------|----------------------------------|
| **Resources**   | Consume GenieACS data read-only                                            | `genieacs://device/{id}`<br>`genieacs://file/{name}`<br>`genieacs://tasks/{id}`<br>`genieacs://devices/list` |
| **Tools**       | Invoke actions on a CPE through GenieACS                                   | `reboot_device`<br>`download_firmware`<br>`refresh_parameter` |

Everything is exposed over a single JSON-RPC endpoint (`/mcp`).  
LLMs / Agents can: `initialize → readResource → listTools → callTool` … and so on.

---

## 🚀 Quick-start (Docker Compose)

Follow instructions from https://github.com/GeiserX/genieacs-container, it is included in the docker compose file there.

## 📦 Install via npm (stdio transport)

```sh
npx genieacs-mcp
```

Or install globally:

```sh
npm install -g genieacs-mcp
genieacs-mcp
```

This downloads the pre-built Go binary for your platform and runs it with stdio transport, compatible with any MCP client.

## 🛠 Local build

```sh
git clone https://github.com/GeiserX/genieacs-mcp
cd genieacs-mcp

# (optional) create .env from the sample
cp .env.example .env && $EDITOR .env

go run ./cmd/server
```

## 🔧 Configuration
| Variable | Default | Description |
|----------|---------|-------------|
| `ACS_URL` | http://localhost:7557 | GenieACS NBI endpoint (without trailing /) |
| `ACS_USER` | admin | GenieACS username |
| `ACS_PASS` | admin | GenieACS password |
| `TRANSPORT` | _(empty = HTTP)_ | Set to `stdio` for stdio transport |

Put them in a `.env` file (from `.env.example`) or set them in the environment. 


## Testing
Tested with [Inspector](https://modelcontextprotocol.io/docs/tools/inspector) and it is currently fully working. Before making a PR, make sure this MCP server behaves well via this medium.

Lacks Testing with actual MCP clients (client LLMs), so please, submit your PRs to improve descriptions in case it fails to adequately match the services offered by this MCP server.

## Example configuration for client LLMs:

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
    "init_method": "initialize",
    "session_header": "Mcp-Session-Id"
  },
  "logo_url": "https://raw.githubusercontent.com/GeiserX/genieacs-container/main/extra/logo.png",
  "contact_email": "acsdesk@protonmail.com",
  "legal_info_url": "https://github.com/GeiserX/genieacs-mcp/blob/main/LICENSE"
}
```

## Credits
[GenieACS](https://github.com/genieacs/genieacs) – the best open-source ACS

[MCP-GO](https://github.com/mark3labs/mcp-go) – modern MCP implementation

[GoReleaser](https://goreleaser.com/) – painless multi-arch releases

## Maintainers

[@GeiserX](https://github.com/GeiserX).

## Contributing

Feel free to dive in! [Open an issue](https://github.com/GeiserX/genieacs-mcp/issues/new) or submit PRs.

GenieACS-MCP follows the [Contributor Covenant](http://contributor-covenant.org/version/2/1/) Code of Conduct.

## GenieACS Ecosystem

This project is part of a broader set of tools for working with GenieACS:

| Project | Type | Description |
|---------|------|-------------|
| [genieacs-docker](https://github.com/GeiserX/genieacs-docker) | Docker + Helm | Production-ready multi-arch Docker image and Helm chart |
| [genieacs-ansible](https://github.com/GeiserX/genieacs-ansible) | Ansible Collection | Dynamic inventory plugin and device management modules |
| [genieacs-ha](https://github.com/GeiserX/genieacs-ha) | HA Integration | Home Assistant integration for TR-069 monitoring |
| [n8n-nodes-genieacs](https://github.com/GeiserX/n8n-nodes-genieacs) | n8n Node | Workflow automation for GenieACS |
| [genieacs-services](https://github.com/GeiserX/genieacs-services) | Service Defs | Systemd/Supervisord service definitions |
| [genieacs-sim-container](https://github.com/GeiserX/genieacs-sim-container) | Simulator | Docker-based GenieACS simulator for testing |

## Other MCP Servers by GeiserX

- [cashpilot-mcp](https://github.com/GeiserX/cashpilot-mcp) — Passive income monitoring
- [duplicacy-mcp](https://github.com/GeiserX/duplicacy-mcp) — Backup health monitoring
- [lynxprompt-mcp](https://github.com/GeiserX/lynxprompt-mcp) — AI configuration blueprints
- [pumperly-mcp](https://github.com/GeiserX/pumperly-mcp) — Fuel and EV charging prices
- [telegram-archive-mcp](https://github.com/GeiserX/telegram-archive-mcp) — Telegram message archive

## Related Projects

| Project | Description |
|---------|-------------|
| [genieacs-container](https://github.com/GeiserX/genieacs-container) | Original and most popular Helm Chart / Container for GenieACS |
| [genieacs-sim-container](https://github.com/GeiserX/genieacs-sim-container) | Docker for the GenieACS Simulator |
| [genieacs-ha](https://github.com/GeiserX/genieacs-ha) | Home Assistant custom integration for GenieACS TR-069 router management |
| [genieacs-ansible](https://github.com/GeiserX/genieacs-ansible) | Ansible Galaxy collection for GenieACS TR-069 ACS |
| [genieacs-services](https://github.com/GeiserX/genieacs-services) | Systemd/Supervisord services for GenieACS processes |
| [n8n-nodes-genieacs](https://github.com/GeiserX/n8n-nodes-genieacs) | n8n community node for GenieACS TR-069 device management |
