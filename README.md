<p align="center">
  <img src="https://raw.githubusercontent.com/GeiserX/genieacs-mcp/main/docs/images/banner.svg" alt="GenieACS MCP" width="900"/>
</p>

<h1 align="center">GenieACS MCP</h1>

<p align="center">
  <a href="https://www.npmjs.com/package/genieacs-mcp"><img src="https://img.shields.io/npm/v/genieacs-mcp?style=flat-square&logo=npm" alt="npm"/></a>
  <a href="https://github.com/GeiserX/genieacs-mcp/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/GeiserX/genieacs-mcp/ci.yml?style=flat-square&logo=github&label=CI" alt="CI"/></a>
  <a href="https://github.com/GeiserX/genieacs-mcp/blob/main/LICENSE"><img src="https://img.shields.io/github/license/GeiserX/genieacs-mcp?style=flat-square" alt="License"/></a>
  <a href="https://hub.docker.com/r/drumsergio/genieacs-mcp"><img src="https://img.shields.io/docker/pulls/drumsergio/genieacs-mcp?style=flat-square&logo=docker" alt="Docker Pulls"/></a>
  <a href="https://github.com/GeiserX/genieacs-mcp/stargazers"><img src="https://img.shields.io/github/stars/GeiserX/genieacs-mcp?style=flat-square&logo=github" alt="GitHub Stars"/></a>
</p>

<p align="center"><strong>A tiny bridge that exposes any GenieACS instance as an MCP v1 (JSON-RPC for LLMs) server written in Go.</strong></p>

LLMs use it to read and manage the CPE devices (routers, ONTs, gateways) that your [GenieACS](https://github.com/genieacs/genieacs) TR-069 server controls, over HTTP or stdio.

## Features

- Read-only resources for devices, files, tasks, presets, provisions and faults (`genieacs://device/{id}`, `genieacs://devices/list`, ...).
- Tools to reboot a device, push firmware, refresh, read and set parameters, and send a connection request.
- Manage presets, provisions and device tags; search devices; delete or retry tasks.
- One JSON-RPC endpoint (`/mcp`) over HTTP, or stdio with `TRANSPORT=stdio`.
- The HTTP transport checks `Host` and `Origin` against DNS rebinding; `MCP_AUTH_TOKEN` adds bearer auth when you expose it.
- Ships as a Docker image, an npm package (`npx genieacs-mcp`) and multi-arch Go binaries.

## Quick start

```sh
npx -y genieacs-mcp
```

The npm package always runs over stdio. In an MCP client that reads an `mcpServers` block, such as Claude Desktop or Cursor:

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

Add `ACS_USER` and `ACS_PASS` to `env` if your NBI needs basic auth. Docker, the HTTP transport and local builds are in [Getting started](https://github.com/GeiserX/genieacs-mcp/blob/main/docs/getting-started.md).

## Documentation

- [Getting started](https://github.com/GeiserX/genieacs-mcp/blob/main/docs/getting-started.md): npm, Docker Compose, local build, first run
- [Configuration](https://github.com/GeiserX/genieacs-mcp/blob/main/docs/configuration.md): environment variables, HTTP security, client config for stdio and HTTP
- [Usage](https://github.com/GeiserX/genieacs-mcp/blob/main/docs/usage.md): resources and tools
- [Development](https://github.com/GeiserX/genieacs-mcp/blob/main/docs/development.md): testing, contributing, credits
- [Related projects](https://github.com/GeiserX/genieacs-mcp/blob/main/docs/related.md): the GenieACS family, other MCP servers, registry listings

## License

[GPL-3.0-or-later](https://github.com/GeiserX/genieacs-mcp/blob/main/LICENSE)
