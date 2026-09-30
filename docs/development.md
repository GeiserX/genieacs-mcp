# Development

<p>
  <a href="https://codecov.io/gh/GeiserX/genieacs-mcp"><img src="https://img.shields.io/codecov/c/github/GeiserX/genieacs-mcp?style=flat-square&logo=codecov&label=Coverage" alt="Coverage"/></a>
</p>

## Testing

Run `go test ./...`. To exercise the server by hand, point
[MCP Inspector](https://modelcontextprotocol.io/docs/tools/inspector) at it: `npx -y
@modelcontextprotocol/inspector -e ACS_URL=http://localhost:7557 npx -y genieacs-mcp`, or connect it to a
running HTTP instance at `http://127.0.0.1:8080/mcp`. The screenshots on [Usage](usage.md) were taken that way
against the demo stack in [Getting started](getting-started.md). A change to a tool's description is a change
to what every client sees; test it with a real client before merging.

## Contributing

[Open an issue](https://github.com/GeiserX/genieacs-mcp/issues/new) or a pull request.

GenieACS-MCP follows the [Contributor Covenant](http://contributor-covenant.org/version/2/1/) Code of Conduct.

## Maintainers

[@GeiserX](https://github.com/GeiserX).

## Credits

[GenieACS](https://github.com/genieacs/genieacs) – the best open-source ACS

[MCP-GO](https://github.com/mark3labs/mcp-go) – modern MCP implementation

[GoReleaser](https://goreleaser.com/) – painless multi-arch releases
