# Ana Board

Ana Board is a small Go-powered split-flap style status board for agents, automations, reminders, CI jobs, and local scripts.

It runs as one local Go server with a browser display, an admin panel, a command-line sender, and a stdio MCP server for tools like Codex, Claude Code, OpenCode, and remote agents such as Hermes.

<p align="center">
  <video src="assets/anaboard-demo.mp4" controls muted playsinline width="900"></video>
</p>

[Watch the demo video](assets/anaboard-demo.mp4)

## About

Ana Board is a private status display for the small signals that should interrupt you gently:

- agent progress updates
- reminders
- email/task nudges
- CI/build/deploy status
- approval requests
- quick personal automations

The usual setup is:

```text
Agent or script
  -> ana-boardctl or ana-board-mcp
  -> private Ana Board HTTP API
  -> browser split-flap display
```

Ana Board is intentionally local-first. Run the display where you can see it, then let remote agents reach it over a private network such as Tailscale. It is not ready to expose as a public unauthenticated web service.

Pieces:

- `ana-board`: local web server, display, admin panel, HTTP API, SSE stream
- `ana-boardctl`: CLI sender for scripts, cron, CI, and remote shells
- `ana-board-mcp`: stdio MCP server for Codex, Claude Code, OpenCode, Hermes-style agents

## Install

Install Go 1.22 or newer, then install the three binaries:

```sh
go install github.com/georgestander/ana-board/cmd/ana-board@latest
go install github.com/georgestander/ana-board/cmd/ana-boardctl@latest
go install github.com/georgestander/ana-board/cmd/ana-board-mcp@latest
```

Make sure Go's bin directory is on `PATH`:

```sh
export PATH="$PATH:$(go env GOPATH)/bin"
```

## Run

```sh
ana-board
```

Open:

```text
http://localhost:8080
http://localhost:8080/admin
```

The server binds to `127.0.0.1:8080` by default. Use another bind address or port explicitly:

```sh
ANA_BOARD_ADDR=127.0.0.1:18080 ana-board
```

From a local clone:

```sh
go run ./cmd/ana-board
```

## Mac + Tailscale + Hermes

Use this when Hermes is running on a Vultr VM and the board display is on your Mac.

1. Install Tailscale on your Mac:

```sh
brew install --cask tailscale
open /Applications/Tailscale.app
```

Log in to Tailscale and make sure your Vultr VM is in the same tailnet.

2. Get your Mac's Tailscale IP:

```sh
tailscale ip -4
```

3. Run Ana Board on that private Tailscale IP:

```sh
ANA_BOARD_ADDR=<mac-tailscale-ip>:18080 ana-board
```

Use the Tailscale IP here, not `127.0.0.1`, when a remote machine such as Vultr needs to connect. Binding to `127.0.0.1` only accepts connections from the Mac itself.

4. Open the board:

```text
http://<mac-tailscale-ip>:18080
http://<mac-tailscale-ip>:18080/admin
```

5. On the Vultr VM, install the sender tools:

```sh
go install github.com/georgestander/ana-board/cmd/ana-boardctl@latest
go install github.com/georgestander/ana-board/cmd/ana-board-mcp@latest
export PATH="$PATH:$(go env GOPATH)/bin"
```

6. Point Hermes at your Mac:

```sh
export ANA_BOARD_URL=http://<mac-tailscale-ip>:18080
ana-boardctl send --source hermes --kind task "[green]HERMES CONNECTED ✅"
```

If Hermes can ping the Mac but `ana-boardctl send` says `connection refused`, Tailscale routing is working but Ana Board is not listening on the Tailscale address. On the Mac, check:

```sh
lsof -nP -iTCP:18080 -sTCP:LISTEN
curl http://<mac-tailscale-ip>:18080/healthz
```

The listener should show `<mac-tailscale-ip>:18080`, not only `127.0.0.1:18080`.

See [docs/hermes-vultr.md](docs/hermes-vultr.md) for the longer setup.

## Send A Message

From the browser, use:

```text
http://localhost:8080/admin
```

From HTTP:

```sh
curl -X POST http://localhost:8080/api/messages \
  -H "Content-Type: application/json" \
  -d '{"text":"[green]B[amber]U[blue]I[violet]L[green]D PASSED ✅","source":"ci","kind":"build"}'
```

From the CLI:

```sh
ana-boardctl capabilities --json
ana-boardctl preview "[blue]HELLO WORLD 🌍"
ana-boardctl send --source codex --kind success "[green]BUILD PASSED ✅"
ana-boardctl send --tiles-json '[{"symbol":"A","color":"green"},{"symbol":"N","color":"amber"},{"symbol":"A","color":"red"}]'
ana-boardctl send --segments-json '[{"text":"ANA ","color":"green"},{"text":"READY ✅","color":"blue"}]'
```

Use a remote/private board URL:

```sh
ANA_BOARD_URL=http://ana-board-host:18080 ana-boardctl send --source hermes "[amber]EMAIL NEEDS REPLY ✉️"
```

## MCP

`cmd/ana-board-mcp` exposes a fixed stdio MCP tool surface:

- `ana_board_capabilities`
- `ana_board_preview_message`
- `ana_board_send_message`
- `ana_board_current`
- `ana_board_recent_messages`
- `ana_board_clear`, requiring `confirm=true`

Claude Code:

```sh
claude mcp add --transport stdio ana-board -- /absolute/path/to/ana-board-mcp
```

Claude Code JSON form:

```sh
claude mcp add-json ana-board '{"type":"stdio","command":"/absolute/path/to/ana-board-mcp","args":[],"env":{"ANA_BOARD_URL":"http://localhost:8080"}}'
```

OpenCode:

```jsonc
{
  "$schema": "https://opencode.ai/config.json",
  "mcp": {
    "ana-board": {
      "type": "local",
      "command": ["/absolute/path/to/ana-board-mcp"],
      "enabled": true,
      "environment": {
        "ANA_BOARD_URL": "http://localhost:8080"
      }
    }
  }
}
```

Hermes on a Vultr VM can run the MCP process locally on that VM and point it at the board over a private Tailscale address:

```sh
ANA_BOARD_URL=http://ana-board-host:18080 ana-board-mcp
```

For the full Vultr setup, see [docs/hermes-vultr.md](docs/hermes-vultr.md).

Do not expose this unauthenticated API with Tailscale Funnel or a public Cloudflare Tunnel yet. Keep it private on localhost or a tightly scoped tailnet.

## Message Limits

- Board size: 6 rows x 22 columns.
- Text is normalized to uppercase.
- Spaces collapse.
- Allowed plain characters: `A-Z`, `0-9`, space, `.`, `,`, `!`, `?`, `:`, `-`, `/`, `'`, `"`.
- Native emoji can be written directly. On Apple devices they render as native iOS/macOS emoji.
- Ana Board does not use an emoji whitelist. The alias list is only a shortcut list.
- Emoji grapheme clusters such as `✅`, `👍🏽`, `🇿🇦`, and `👨‍👩‍👧‍👦` count as one board tile.
- Common named aliases like `:rocket:`, `:check:`, `:warning:`, `:mail:`, `:calendar:`, and `:globe:` also work.
- Unknown named aliases are treated as plain text, so agents are safest when they send native emoji directly.
- Messages that need more than 6 rows fail.
- Words longer than 22 tiles fail.
- Exact per-tile color can be sent with JSON `tiles`: `[{"symbol":"A","color":"green"},{"symbol":"N","color":"amber"},{"symbol":"A","color":"red"}]`.
- Quick text can color individual letters with inline tokens: `[green]A[amber]N[red]A [blue]READY`.
- Agents can also send JSON `segments` when phrases share a color: `[{"text":"OK ","color":"green"},{"text":"FAIL","color":"red"}]`.
- The `color` metadata field is only the default for tiles without an inline token or segment color.

Metadata:

- `animation`: `row`
- `color`: default/fallback tile color, one of `white`, `green`, `amber`, `red`, `blue`, `violet`
- `kind`: `info`, `success`, `warning`, `error`, `reminder`, `email`, `task`, `deploy`, `build`
- `priority`: `low`, `normal`, `high`

## API

```text
GET  /healthz
GET  /
GET  /admin
POST /admin/messages
POST /admin/clear
GET  /api/current
GET  /api/messages
POST /api/messages
POST /api/clear
GET  /events
```

## Test

```sh
go test ./...
go vet ./...
node --check web/static/board.js
node --check web/static/admin.js
```

If Go tries to write its build cache outside the sandboxed workspace:

```sh
GOCACHE=$PWD/.gocache go test ./...
```
