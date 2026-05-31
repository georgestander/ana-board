# Agent Integrations

Ana Board has two agent-facing paths:

1. `ana-boardctl` for shells, cron jobs, GitHub Actions, and simple remote scripts.
2. `ana-board-mcp` for agent clients that understand MCP tools.

Both paths call the same Ana Board HTTP API. Keep the board server private unless API token auth is added later.

## Recommended Flow

Agents should:

1. Read capabilities first.
2. Send concise status updates.
3. Prefer one useful native emoji, not decoration.
4. Use per-tile color as signal with JSON `tiles`, inline color tokens, or JSON segments.
5. Use native emoji directly when useful; there is no emoji whitelist.
6. Avoid sending secrets, raw email bodies, or private customer text.
7. Use `row` animation only.

Good messages:

```text
[green]BUILD PASSED ✅
[amber]EMAIL NEEDS REPLY ✉️
[blue]DEPLOY COMPLETE 🚀
[red]AGENT NEEDS APPROVAL ⚠️
```

The optional `emoji_aliases` list is only a shortcut list for common symbols. Agents can send native Unicode emoji directly for the full iOS/macOS emoji range. The `color` field is only a default. Use `tiles` when individual letters should have different colors, `[green]` style inline tokens for quick text, or `segments` when phrases share a color.

Exact per-letter color:

```json
{
  "tiles": [
    {"symbol":"A","color":"green"},
    {"symbol":"N","color":"amber"},
    {"symbol":"A","color":"red"}
  ]
}
```

## Hermes On Vultr

Install the public tools on the Vultr VM:

```sh
go install github.com/georgestander/ana-board/cmd/ana-boardctl@latest
go install github.com/georgestander/ana-board/cmd/ana-board-mcp@latest
```

Run Tailscale on the board host and the Vultr VM. Then run the MCP server or CLI on the Vultr VM with:

```sh
ANA_BOARD_URL=http://ana-board-host:18080 ana-board-mcp
```

or:

```sh
ANA_BOARD_URL=http://ana-board-host:18080 ana-boardctl send --source hermes --kind task "[amber]TASK NEEDS REVIEW 📌"
```

This keeps the board private inside the tailnet. Do not use Tailscale Funnel for this unauthenticated build.

If the Vultr agent can ping the board host but `ana-boardctl send` gets `connection refused`, the board host is reachable but Ana Board is not listening on the Tailscale IP. Restart the board host with `ANA_BOARD_ADDR=<board-tailscale-ip>:18080 ana-board` and verify with `curl http://<board-tailscale-ip>:18080/healthz`.

See [hermes-vultr.md](hermes-vultr.md) for the full setup.

## Codex

Use the CLI from a repo or script:

```sh
ana-boardctl send --source codex --kind task "[blue]CODEX REVIEWING 💻"
```

Use MCP when Codex has a local MCP configuration that can launch `ana-board-mcp`.

## Claude Code

Register the MCP server:

```sh
claude mcp add --transport stdio ana-board -- /absolute/path/to/ana-board-mcp
```

For a remote board over Tailscale:

```sh
claude mcp add-json ana-board '{"type":"stdio","command":"/absolute/path/to/ana-board-mcp","args":[],"env":{"ANA_BOARD_URL":"http://ana-board-host:18080"}}'
```

## OpenCode

Add this to `opencode.json`:

```jsonc
{
  "$schema": "https://opencode.ai/config.json",
  "mcp": {
    "ana-board": {
      "type": "local",
      "command": ["/absolute/path/to/ana-board-mcp"],
      "enabled": true,
      "environment": {
        "ANA_BOARD_URL": "http://ana-board-host:18080"
      }
    }
  }
}
```

## Safety Notes

- Default server bind is localhost.
- Tailscale access is still remote access.
- Do not expose the write API publicly until auth exists.
- `ana_board_clear` requires `confirm=true`.
- MCP tools are fixed; the MCP server does not proxy arbitrary HTTP paths.
