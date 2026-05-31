# Hermes On Vultr

This is the recommended setup for a Hermes agent running on a Vultr VM.

Ana Board should run where you can see it, usually on your Mac or local display machine. Hermes should send updates to that local board over a private Tailscale network.

Do not expose Ana Board's write API on the public internet until API token auth exists.

## 1. Install Ana Board Tools On Vultr

Install Go 1.22 or newer, then install the sender tools from the public repo:

```sh
go install github.com/georgestander/ana-board/cmd/ana-boardctl@latest
go install github.com/georgestander/ana-board/cmd/ana-board-mcp@latest
```

Make sure Go's bin directory is on `PATH`:

```sh
export PATH="$PATH:$(go env GOPATH)/bin"
```

## 2. Run The Board On Your Local Machine

Join both your local machine and the Vultr VM to the same Tailscale tailnet.

On the local machine, get the Tailscale IP:

```sh
tailscale ip -4
```

Run Ana Board on that private address:

```sh
ANA_BOARD_ADDR=<local-tailscale-ip>:18080 ana-board
```

Do not bind to `127.0.0.1` for this setup. `127.0.0.1` only accepts connections from the local machine, so Hermes on Vultr will get `connection refused` even if Tailscale ping works.

Then open the display locally:

```text
http://<local-tailscale-ip>:18080
http://<local-tailscale-ip>:18080/admin
```

If you are running from a clone instead of an installed binary:

```sh
ANA_BOARD_ADDR=<local-tailscale-ip>:18080 go run ./cmd/ana-board
```

## 3. Point Hermes At The Board

On Vultr:

```sh
export ANA_BOARD_URL=http://<local-tailscale-ip>:18080
```

Smoke test:

```sh
ana-boardctl capabilities --json
ana-boardctl send --source hermes --kind task "[amber]HERMES CONNECTED 📌"
```

If the smoke test fails with `connect: connection refused`, check the Mac-side listener:

```sh
lsof -nP -iTCP:18080 -sTCP:LISTEN
curl http://<local-tailscale-ip>:18080/healthz
```

The listener must be on `<local-tailscale-ip>:18080`. If it shows `127.0.0.1:18080`, stop Ana Board and restart it with:

```sh
ANA_BOARD_ADDR=<local-tailscale-ip>:18080 ana-board
```

If ping works but `curl` times out instead of refusing immediately, check the Mac firewall or Tailscale ACLs.

## 4. MCP Mode

If Hermes can launch MCP tools, configure it to run:

```sh
ANA_BOARD_URL=http://<local-tailscale-ip>:18080 ana-board-mcp
```

Hermes should call `ana_board_capabilities` first, then `ana_board_send_message` for meaningful notifications.

## 5. Message Guidance

Good notification examples:

```text
[green]TASK COMPLETE ✅
[amber]EMAIL NEEDS REPLY ✉️
[red]APPROVAL NEEDED ⚠️
[blue]REMINDER DUE 📅
```

Keep messages short. Do not send secrets, raw email bodies, customer text, tokens, or long logs to the board.
