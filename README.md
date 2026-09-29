<p align="center">
  <img src="https://raw.githubusercontent.com/GeiserX/genieacs-mcp/main/docs/images/banner.svg" alt="GenieACS MCP banner" width="900"/>
</p>

<h1 align="center">GenieACS-MCP</h1>

<p align="center">
  <a href="https://www.npmjs.com/package/genieacs-mcp"><img src="https://img.shields.io/npm/v/genieacs-mcp?style=flat-square&logo=npm" alt="npm"/></a>
  <a href="https://github.com/GeiserX/genieacs-mcp/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/GeiserX/genieacs-mcp/ci.yml?style=flat-square&logo=github&label=CI" alt="CI"/></a>
  <a href="https://hub.docker.com/r/drumsergio/genieacs-mcp"><img src="https://img.shields.io/docker/pulls/drumsergio/genieacs-mcp?style=flat-square&logo=docker" alt="Docker Pulls"/></a>
  <a href="https://github.com/GeiserX/genieacs-mcp/stargazers"><img src="https://img.shields.io/github/stars/GeiserX/genieacs-mcp?style=flat-square&logo=github" alt="GitHub Stars"/></a>
  <a href="https://github.com/GeiserX/genieacs-mcp/blob/main/LICENSE"><img src="https://img.shields.io/github/license/GeiserX/genieacs-mcp?style=flat-square" alt="License"/></a>
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
npx genieacs-mcp
```

Set `ACS_URL` (and `ACS_USER` / `ACS_PASS` if your NBI needs basic auth) first. For Docker, the server is already in the compose file of [genieacs-container](https://github.com/GeiserX/genieacs-container). Local builds are in [Installation](https://github.com/GeiserX/genieacs-mcp/blob/main/docs/installation.md).

## Documentation

- [Installation](https://github.com/GeiserX/genieacs-mcp/blob/main/docs/installation.md): Docker Compose, npm, local build
- [Configuration](https://github.com/GeiserX/genieacs-mcp/blob/main/docs/configuration.md): environment variables, HTTP security and an example client config
- [Resources and tools](https://github.com/GeiserX/genieacs-mcp/blob/main/docs/usage.md)
- [Development](https://github.com/GeiserX/genieacs-mcp/blob/main/docs/development.md): testing, contributing, credits
- [Related projects and listings](https://github.com/GeiserX/genieacs-mcp/blob/main/docs/related.md): the GenieACS ecosystem and other MCP servers

## License

[GPL-3.0](https://github.com/GeiserX/genieacs-mcp/blob/main/LICENSE)
