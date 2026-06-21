# @apresai/gimage-mcp

MCP (Model Context Protocol) server for AI-powered image generation and processing,
backed by the [`gimage`](https://github.com/apresai/gimage) CLI. Exposes image
generation (Gemini, Vertex AI, xAI Grok) plus resize/scale/crop/compress/
convert and batch operations as MCP tools for Claude Desktop and other MCP clients.

## Install

```bash
npm install -g @apresai/gimage-mcp
```

This package is a thin Node wrapper. On install it downloads the platform-native
`gimage` binary from the matching GitHub release into the package. If your package
manager blocks postinstall scripts, the binary is fetched automatically on first run,
or it falls back to a `gimage` already on your `PATH` (e.g. installed via Homebrew:
`brew install apresai/tap/gimage`).

## Use with Claude Desktop

Add to your MCP configuration
(`~/Library/Application Support/Claude/claude_desktop_config.json` on macOS):

```json
{
  "mcpServers": {
    "gimage": {
      "command": "npx",
      "args": ["-y", "@apresai/gimage-mcp"]
    }
  }
}
```

Running `gimage-mcp` with no arguments starts the MCP server over stdio. Any arguments
are forwarded to the underlying `gimage` CLI, so `gimage-mcp --version` prints the
version and `gimage-mcp serve` is equivalent to the default.

## Tools

`generate_image`, `resize_image`, `scale_image`, `crop_image`, `compress_image`,
`convert_image`, `batch_resize`, `batch_compress`, `batch_convert`, `list_models`.

## Configure API keys

```bash
gimage auth setup gemini
```

See the [main project README](https://github.com/apresai/gimage#readme) for full
documentation, supported models, and pricing.

## License

MIT
