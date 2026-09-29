# Installation

## Docker Compose

Follow instructions from https://github.com/GeiserX/genieacs-container, it is included in the docker compose file there.

## npm (stdio transport)

```sh
npx genieacs-mcp
```

Or install globally:

```sh
npm install -g genieacs-mcp
genieacs-mcp
```

This downloads the pre-built Go binary for your platform and runs it with stdio transport, compatible with any MCP client.

## Local build

```sh
git clone https://github.com/GeiserX/genieacs-mcp
cd genieacs-mcp

# (optional) create .env from the sample
cp .env.example .env && $EDITOR .env

go run ./cmd/server
```
