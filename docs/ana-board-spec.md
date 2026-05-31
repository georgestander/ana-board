# Ana Board: Go Learning Build Spec

Status: Draft v0.1  
Date: 2026-05-30  
Working name: Ana Board  
Alternate names: AgentBoard, OpenFlap, SignalBoard, Ambient Agent Board

## 1. What We Are Building

Ana Board is a self-hosted split-flap style status board for AI agents, scripts, CI jobs, reminders, and home automations.

The simplest version is:

```text
An outside system sends a message.
The Go server accepts it.
The layout engine turns it into a fixed board frame.
The browser display receives the frame.
The tiles animate into place.
```

The product should feel like a digital Vestaboard-inspired ambient display, but the purpose is different:

> A quiet wall display where agents, automations, and tools can surface useful status without needing a full dashboard.

This project is also a learning vehicle for Go. The build should be paced so each feature introduces one primitive concept clearly.

## 2. Why This Is a Good Go Project

Go is good for this project because the system is mostly:

- small data structures
- clear transformations
- HTTP endpoints
- streaming updates
- simple concurrency
- explicit error handling
- deployment as a single binary

That means we can learn Go from the ground up while building something real.

The goal is not to become a low-level coder first. The goal is to understand the system pieces well enough to architect it, guide Codex, review decisions, and know when a design feels too complicated.

## 3. Learning Philosophy

You are learning as the architect and systems engineer.

That means every phase should answer four questions:

1. What is the primitive system concept?
2. What Go feature represents that concept?
3. What product behavior does it unlock?
4. How do we verify that it works?

Avoid learning Go as a bag of syntax. Learn it as a way to assemble small machines.

The recurring mental model:

```text
Data shape -> function -> result
Data shape -> method -> behavior
Request -> handler -> response
Event -> channel -> subscriber
Interface -> implementation -> swappable system boundary
```

### 3.1 Go From Zero Primer

This section is for reading the code like an architect, even before the syntax feels familiar.

Go programs are built from a small number of primitive puzzle pieces:

| Puzzle piece | Plain-English meaning | Ana Board example |
| --- | --- | --- |
| package | A named box of related code | `board`, `layout`, `server` |
| file | A place to keep related definitions | `frame.go`, `center.go` |
| variable | A named value that can change | `currentFrame` |
| constant | A named value that should not change | `DefaultRows = 6` |
| type | A named shape of data | `Frame`, `Message`, `Cell` |
| struct | A bundle of named fields | a `Message` has text, source, priority |
| function | A transformation or action | `NormalizeText(input)` |
| method | A function attached to a type | `frame.Set(row, col, cell)` |
| interface | A promise about behavior | anything that can `SaveMessage` is a `Store` |
| error | An explicit failure value | message does not fit on board |
| test | A small proof of behavior | centered text lands where expected |
| goroutine | Work running alongside other work | broadcast frame updates |
| channel | A pipe for passing values | send frames to subscribers |
| context | Request lifetime and cancellation | stop streaming when browser disconnects |

How to read a Go file:

1. Look at the `package` line first. That tells you which box you are in.
2. Look at the `import` block. That tells you what outside tools this file uses.
3. Look for `type` definitions. These are the nouns of the system.
4. Look for functions and methods. These are the verbs.
5. Look for returned `error` values. These are the rules and failure boundaries.
6. Look at tests to understand the intended behavior faster than reading implementation details.

Example file:

```go
package board

import "fmt"

const (
    DefaultRows = 6
    DefaultCols = 22
    DefaultColor = "white"
)

type Cell struct {
    Symbol string
    Color  string
}

type Frame struct {
    Rows  int
    Cols  int
    Cells [][]Cell
}

func NewFrame(rows, cols int) (Frame, error) {
    if rows <= 0 {
        return Frame{}, fmt.Errorf("rows must be positive")
    }

    if cols <= 0 {
        return Frame{}, fmt.Errorf("cols must be positive")
    }

    cells := make([][]Cell, rows)
    for row := range cells {
        cells[row] = make([]Cell, cols)
        for col := range cells[row] {
            cells[row][col] = Cell{Symbol: " ", Color: DefaultColor}
        }
    }

    return Frame{
        Rows:  rows,
        Cols:  cols,
        Cells: cells,
    }, nil
}

func (f Frame) Set(row, col int, cell Cell) (Frame, error) {
    if row < 0 || row >= f.Rows {
        return f, fmt.Errorf("row %d is outside frame", row)
    }

    if col < 0 || col >= f.Cols {
        return f, fmt.Errorf("col %d is outside frame", col)
    }

    f.Cells[row][col] = cell
    return f, nil
}
```

How to read that file:

1. `package board`

   This file belongs to the `board` box. It should only contain board-domain ideas: cells, frames, dimensions, and board rules. If this file starts handling HTTP or Cloudflare, something has drifted.

2. `import "fmt"`

   This file uses one outside standard-library tool: `fmt`. Here it is only used to create readable errors. No database, no web server, no frontend dependency. That is a clue that this file is still core engine code.

3. `const DefaultRows = 6`

   Constants name facts that should not casually change while the program runs. The 6 x 22 board shape becomes a named product rule instead of a mystery number scattered through the code.

4. `type Cell struct`

   This is a noun. A `Cell` is one position on the board. It has a visible `Symbol` and a `Color`.

5. `type Frame struct`

   This is another noun. A `Frame` is the whole board at one moment. Its job is to hold the exact rows, columns, and cells that should appear on screen.

6. `func NewFrame(...)`

   This is a constructor function. Its job is to create a valid `Frame`. Notice that it returns `(Frame, error)`, which means construction can fail.

7. `if rows <= 0`

   This is a boundary rule. The system refuses to create nonsense frames. That is what "make invalid states hard to represent" looks like in code.

8. `make([][]Cell, rows)`

   This creates the outer slice: the rows. A slice is Go's flexible list-like collection.

9. `make([]Cell, cols)`

   This creates each row's cells. The nested structure mirrors the board shape:

   ```text
   rows
     row
       cells
   ```

10. `Cell{Symbol: " ", Color: DefaultColor}`

    Every empty cell starts as a visible space. That means a new frame is blank, not random or undefined.

11. `func (f Frame) Set(...)`

    This is a method because it has a receiver: `(f Frame)`. Plain-English reading: "A Frame knows how to set one of its cells."

12. `return f, fmt.Errorf(...)`

    These are failure boundaries. The method does not silently ignore a bad row or column. It returns the original frame plus a useful error.

Example test:

```go
package board

import "testing"

func TestNewFrameCreatesBlankFrame(t *testing.T) {
    frame, err := NewFrame(6, 22)
    if err != nil {
        t.Fatalf("NewFrame returned error: %v", err)
    }

    if frame.Rows != 6 {
        t.Fatalf("Rows = %d, want 6", frame.Rows)
    }

    if frame.Cols != 22 {
        t.Fatalf("Cols = %d, want 22", frame.Cols)
    }

    if len(frame.Cells) != 6 {
        t.Fatalf("len(Cells) = %d, want 6", len(frame.Cells))
    }

    if len(frame.Cells[0]) != 22 {
        t.Fatalf("len(Cells[0]) = %d, want 22", len(frame.Cells[0]))
    }

    if frame.Cells[0][0].Symbol != " " {
        t.Fatalf("first cell = %q, want space", frame.Cells[0][0].Symbol)
    }
}

func TestNewFrameRejectsInvalidDimensions(t *testing.T) {
    _, err := NewFrame(0, 22)
    if err == nil {
        t.Fatal("expected error for zero rows")
    }
}
```

How to read that test:

1. `package board`

   The test is in the same package as the code. It is testing the board engine directly.

2. `func TestNewFrameCreatesBlankFrame(t *testing.T)`

   Test functions are named examples of expected behavior. This one says: creating a frame should produce a blank frame.

3. `frame, err := NewFrame(6, 22)`

   This calls the real constructor. The test starts with the behavior a user of the package would use.

4. `if err != nil`

   Before inspecting the result, the test checks whether construction failed.

5. `t.Fatalf(...)`

   This stops the test immediately with a useful explanation. Tests are not just pass/fail switches; they are documentation for future debugging.

6. `want 6`, `want 22`, `want space`

   This is the most useful reading shortcut. In tests, look for the `want` values. They tell you the intended behavior without needing to understand every line of implementation.

7. `TestNewFrameRejectsInvalidDimensions`

   This test names a rule: invalid dimensions are rejected. Tests often reveal the system's rules more clearly than production code.

Architect reading shortcut:

```text
Production file:
What nouns and verbs exist?
What errors define the boundaries?

Test file:
What behavior is promised?
What inputs are allowed?
What inputs are rejected?
```

How to think about `internal/`:

```text
Code inside internal/ is application-private.
Other projects cannot import it.
That makes it a good place for the real Ana Board engine.
```

How to think about `cmd/`:

```text
cmd/ana-board is the executable entrypoint.
It should wire pieces together.
It should not contain most business logic.
```

The first big architectural split:

```text
cmd/ana-board
  starts the app
  reads config
  wires server + store + layout

internal/board
  knows what a board frame is

internal/layout
  knows how text becomes a board frame

internal/server
  knows how HTTP talks to the app
```

This is the system-engineer view of Go: each package owns one part of the machine.

## 4. Product Principle

Ana Board should remain charming because it is constrained.

The board has fixed space. Messages must fit. Animations should feel physical. The API should stay tiny.

The constraint is the feature.

Do not turn v0 into a generic dashboard, notification center, plugin platform, or account system.

## 5. First Product Slice

The first useful slice is:

```text
Run locally:
go run ./cmd/ana-board

Open:
http://localhost:8080

Send:
curl -X POST http://localhost:8080/api/messages \
  -H "Content-Type: application/json" \
  -d '{"text":"HELLO FROM CODEX","source":"curl"}'

Result:
Browser board flips to the new message.
```

Everything else comes after that.

## 6. Board Model

The board is a fixed grid.

Default dimensions:

- columns: 22
- rows: 6
- total cells: 132

The key primitives are:

```text
Cell
Frame
Message
Board
Layout
Animation
Store
Stream
```

### 6.1 Cell

A cell is one visible position on the board.

Conceptually:

```go
type Cell struct {
    Symbol string
    Color string
}
```

Learning concept:

- `struct`
- field names
- zero values
- why emoji need a string symbol instead of a single rune

Product concept:

- every tile has one final visible symbol
- native emoji such as `🚀`, `👍🏽`, and `🇿🇦` still occupy one tile
- named aliases such as `:rocket:` are convenience input, not the full emoji set
- colors are tile metadata and may be assigned per tile or segment

### 6.2 Frame

A frame is the complete board state at one moment.

Conceptually:

```go
type Frame struct {
    Rows  int
    Cols  int
    Cells [][]Cell
}
```

Learning concept:

- slices
- nested slices
- constructors
- invariants

Product concept:

- the browser does not receive vague text
- it receives exact board state

Important invariant:

```text
A frame must always contain exactly Rows x Cols cells.
```

Invalid states should be hard to represent.

### 6.3 Message

A message is what an external actor submits.

Conceptually:

```go
type Message struct {
    ID        string
    Text      string
    Source    string
    Priority  string
    Animation string
    CreatedAt time.Time
}
```

Learning concept:

- structs with multiple field types
- standard library `time`
- validation
- JSON tags later

Product concept:

- humans and agents submit messages
- the system records where they came from

### 6.4 Layout

Layout is the transformation from message text into a frame.

Conceptually:

```text
"HELLO FROM CODEX"
        |
        v
6 rows x 22 columns
        |
        v
centered frame
```

Learning concept:

- functions
- strings
- loops
- slices
- errors
- tests

Product concept:

- messages become board-ready
- formatting rules live in one place

## 7. Character Set

Start with a deliberately small character set:

```text
A-Z
0-9
space
basic punctuation: . , ! ? : - / ' "
```

Later:

- color blocks
- arrows
- symbols
- emoji approximation
- custom palettes

Rules for v0:

- convert lowercase letters to uppercase
- collapse unsupported characters to a space or `?`
- trim excessive whitespace
- fail loudly if the message is empty after normalization

Learning concept:

- string normalization
- input validation
- explicit product rules

## 8. Layout Modes

Start with one layout mode:

```text
center
```

Then add:

- left
- wrap
- title-footer
- exact-frame

### 8.1 Center Layout

Center layout should:

- split text into words
- wrap lines to 22 columns
- center each line horizontally
- center all lines vertically within 6 rows
- return an error if the text cannot fit

Example:

```text
Input:
HELLO FROM CODEX

Output:
                      
                      
   HELLO FROM CODEX  
                      
                      
                      
```

### 8.2 Wrap Layout

Wrap layout should:

- use as many rows as needed up to 6
- preserve word boundaries where possible
- fail if any single word is longer than 22 characters unless a breaking strategy is supplied

### 8.3 Exact Frame Layout

Exact frame layout is for advanced users and agents.

They can submit a full 6 x 22 frame directly.

This should not be part of the first milestone, but the architecture should leave room for it.

## 9. System Architecture

The system should begin as a single Go binary.

```text
cmd/ana-board/main.go
        |
        v
internal/server
        |
        +--> internal/board
        +--> internal/layout
        +--> internal/store
        +--> internal/stream
```

Initial repo shape:

```text
ana-board/
  cmd/
    ana-board/
      main.go
  internal/
    board/
      cell.go
      frame.go
      charset.go
      frame_test.go
    layout/
      center.go
      wrap.go
      layout_test.go
    messages/
      message.go
      validation.go
      validation_test.go
    store/
      store.go
      memory.go
      memory_test.go
    stream/
      broker.go
      broker_test.go
    server/
      server.go
      routes.go
      handlers.go
      sse.go
  web/
    static/
      board.css
      board.js
  docs/
    ana-board-spec.md
  go.mod
  README.md
```

Do not add templ, htmx, D1, Cloudflare, or auth until the local vertical slice works.

## 10. Data Flow

### 10.1 Local v0 Flow

```text
POST /api/messages
        |
        v
decode JSON
        |
        v
validate Message
        |
        v
layout text into Frame
        |
        v
save message in MemoryStore
        |
        v
set current frame
        |
        v
broadcast frame to SSE subscribers
        |
        v
browser animates board
```

### 10.2 Later Cloudflare Flow

```text
Browser or agent
        |
        v
Cloudflare Worker
        |
        +--> token check
        +--> D1 write
        +--> forward to Go container
                 |
                 v
              layout + stream
                 |
                 v
              browser display
```

Important principle:

The Go app should not care whether persistence is memory, SQLite, or D1.

That is why we introduce a Store interface.

## 11. Store Interface

The store is the memory of the system.

Conceptually:

```go
type Store interface {
    SaveMessage(ctx context.Context, msg Message) error
    ListMessages(ctx context.Context, limit int) ([]Message, error)
    CurrentFrame(ctx context.Context, boardID string) (Frame, error)
    SaveCurrentFrame(ctx context.Context, boardID string, frame Frame) error
}
```

Learning concept:

- interfaces
- dependency inversion
- context
- method signatures
- implementations

Product concept:

- start simple with memory
- later swap in D1 or SQLite
- keep business logic independent from storage

Initial implementation:

```text
MemoryStore
```

Later implementations:

```text
SQLiteStore
D1BridgeStore
```

## 12. Server API

### 12.1 POST /api/messages

Submit a message.

Request:

```json
{
  "text": "[green]B[amber]U[blue]I[violet]L[green]D FINISHED ✅",
  "source": "codex",
  "priority": "normal",
  "animation": "row"
}
```

Response:

```json
{
  "id": "msg_123",
  "status": "displayed",
  "frame": {
    "rows": 6,
    "cols": 22,
    "cells": []
  }
}
```

Validation:

- `text` is required
- `source` defaults to `unknown`
- `priority` defaults to `normal`
- `animation` defaults to `row`; row is the only supported animation
- exact per-tile color can be sent with `tiles`, for example `[{"symbol":"A","color":"green"},{"symbol":"N","color":"amber"}]`
- message must fit the selected layout

Learning concept:

- HTTP handlers
- JSON decode
- JSON encode
- status codes
- request validation

### 12.2 GET /api/current

Return the current frame.

Response:

```json
{
  "board_id": "default",
  "frame": {
    "rows": 6,
    "cols": 22,
    "cells": []
  },
  "updated_at": "2026-05-30T10:00:00Z"
}
```

### 12.3 GET /api/messages

Return recent messages.

Query parameters:

- `limit`, default 20

### 12.4 GET /events

Open an SSE stream.

Event:

```text
event: frame
data: {"rows":6,"cols":22,"cells":[...],"animation":"row"}
```

Learning concept:

- streaming HTTP responses
- flushing
- goroutines
- channels
- client disconnects
- context cancellation

## 13. UI Requirements

The first UI should be a fullscreen display.

Routes:

```text
GET /
```

It should:

- render a 22 x 6 grid
- connect to `/events` with `EventSource`
- animate tile changes
- load `/api/current` on first page load
- work without React

Use:

- server-rendered HTML
- plain CSS
- small vanilla JavaScript file

Avoid:

- React
- frontend build tooling
- complex state management
- decorative dashboard layout

The UI is not a landing page. It is the board.

## 14. Split-Flap Animation

Animation should live in the browser.

The server sends the final frame. The browser decides how to visually transition from old frame to new frame.

Initial animation mode:

- row
- source-specific styles

Learning concept:

- the server owns truth
- the browser owns presentation
- separating state from animation keeps the backend simple

## 15. Message Queue

Do not build a full queue in the first slice.

Start with:

```text
new message replaces current board
```

Then add:

```text
new message enters queue
board displays one message at a time
priority messages can interrupt
```

Queue concepts:

- pending
- displayed
- skipped
- expired

Potential message statuses:

```text
queued
displayed
expired
cancelled
failed
```

Learning concept:

- simple state machines
- slices as queues
- goroutines for background workers
- time-based behavior

## 16. Auth

No auth in milestone 1.

Add auth only when exposing the API beyond localhost.

Auth v1:

- static API token from environment variable
- clients send `Authorization: Bearer <token>`
- reject missing or invalid tokens

Auth later:

- D1-backed token table
- hashed tokens
- token names
- revocation
- per-source permissions

Learning concept:

- environment variables
- middleware
- security boundaries
- token hashing later

## 17. Persistence

Persistence phases:

1. Memory only
2. Local SQLite
3. Cloudflare D1 through Worker bridge

Do not start with D1.

Reason:

D1 is useful later, but it adds Cloudflare concepts before the Go primitives are understood.

The learning-friendly route is:

```text
MemoryStore first.
Store interface second.
SQLite or D1 later.
```

## 18. Cloudflare Deployment

Cloudflare should be a later phase.

Target architecture:

```text
Cloudflare Worker
        |
        +--> D1
        |
        +--> Cloudflare Container
                 |
                 v
              Go binary
```

The Worker handles:

- public entrypoint
- request routing
- D1 binding
- optional API token checks

The Go container handles:

- board model
- layout engine
- API semantics
- SSE display stream
- HTML rendering

The Go app remains the real application.

## 19. Build Phases

### Phase 0: Project Skeleton

Goal:

Create a minimal Go project with no external dependencies.

Files:

```text
go.mod
cmd/ana-board/main.go
internal/board
internal/layout
README.md
```

Go concepts:

- modules
- packages
- `main`
- imports
- tests

Done when:

- `go test ./...` passes
- `go run ./cmd/ana-board` prints or serves a tiny health response

Architect questions:

- What is the smallest useful binary?
- What belongs in `main`?
- What belongs in internal packages?

### Phase 1: Board Engine

Goal:

Represent a 6 x 22 board frame safely.

Build:

- `Cell`
- `Frame`
- `NewFrame(rows, cols int)`
- `Set(row, col int, cell Cell)`
- `Get(row, col int)`
- frame serialization shape

Go concepts:

- structs
- methods
- slices
- constructors
- errors
- table-driven tests

Done when:

- invalid dimensions fail
- out-of-bounds cells fail
- empty frames are always exactly 6 x 22
- tests cover row/column edge cases

### Phase 2: Character Set and Normalization

Goal:

Convert user text into board-safe text.

Build:

- allowed character set
- `NormalizeText`
- uppercase conversion
- unsupported character handling
- empty message validation

Go concepts:

- strings
- runes
- maps
- loops
- error values

Done when:

- `"hello"` becomes `"HELLO"`
- unsupported characters are handled consistently
- empty or whitespace-only input fails

### Phase 3: Layout Engine

Goal:

Turn text into a centered frame.

Build:

- word wrapping
- horizontal centering
- vertical centering
- fit errors

Go concepts:

- pure functions
- slices
- helper functions
- testing behavior instead of implementation

Done when:

- short messages center correctly
- multi-line messages wrap correctly
- too-long messages return useful errors
- all layout output is exactly 6 x 22

### Phase 4: HTTP Server

Goal:

Expose the board over local HTTP.

Build:

- `GET /healthz`
- `POST /api/messages`
- `GET /api/current`
- JSON request/response types
- useful error responses

Go concepts:

- `net/http`
- handlers
- JSON encode/decode
- status codes
- request body limits

Done when:

- `curl` can submit a message
- `curl` can read current frame
- malformed JSON returns 400
- oversized messages return clear errors

### Phase 5: Memory Store

Goal:

Keep current board state and recent messages in memory.

Build:

- `Store` interface
- `MemoryStore`
- current frame storage
- recent messages list

Go concepts:

- interfaces
- pointers
- mutexes if needed
- context-aware method signatures

Done when:

- messages are stored
- current frame can be read
- tests prove store behavior

### Phase 6: Browser Display

Goal:

Show the board in a browser.

Build:

- `GET /`
- HTML template or literal template file
- 22 x 6 grid
- static CSS
- static JavaScript
- fetch current frame on load

Go concepts:

- serving static files
- HTML responses
- filesystem layout
- maybe `embed`

Frontend concepts:

- CSS grid
- DOM updates
- fixed aspect ratio
- responsive sizing

Done when:

- opening localhost shows the board
- submitting a message updates after refresh
- layout remains stable on desktop and mobile widths

### Phase 7: Live Updates with SSE

Goal:

The browser updates without refresh.

Build:

- `GET /events`
- SSE broker
- subscriber channels
- broadcast new frames
- browser `EventSource`

Go concepts:

- channels
- goroutines
- context cancellation
- HTTP flushing
- client disconnects

Done when:

- browser receives new frames live
- closing browser removes subscriber
- multiple browser tabs receive updates

### Phase 8: Animation

Goal:

Make the board feel like a split-flap display.

Build:

- tile state in JS
- row transition
- CSS flip effect

Backend rule:

The Go server sends target state only.

Frontend rule:

The browser performs the animation.

Done when:

- character changes animate
- unchanged cells do not distract
- animation does not break layout

### Phase 9: Admin Panel

Goal:

Let a human send test messages and inspect recent history.

Build:

- `GET /admin`
- form to submit message
- recent messages list
- clear current frame button

Potential tools:

- plain HTML first
- htmx later
- templ later

Go concepts:

- form handling
- content types
- redirect vs partial response

Done when:

- user can submit from browser
- history is visible
- API remains usable by agents

### Phase 10: CLI

Goal:

Create a tiny command-line sender that agents and scripts can understand.

Command:

```text
ana-boardctl send "[green]BUILD PASSED ✅"
```

Build:

- `cmd/ana-boardctl`
- config via environment variables
- HTTP client request
- `capabilities --json`
- `preview`
- clear confirmation

Go concepts:

- command-line args
- `flag`
- HTTP clients
- environment variables
- shared client package

Done when:

- CLI sends to local server
- CLI previews emoji/color frames without sending
- `capabilities --json` explains rows, columns, colors, kinds, animations, native emoji support, and optional aliases
- failures print clear errors

### Phase 10.5: MCP Server

Goal:

Expose Ana Board as a fixed MCP stdio tool server.

Build:

- `cmd/ana-board-mcp`
- JSON-RPC over stdio
- `initialize`
- `tools/list`
- `tools/call`
- fixed tools only:
  - `ana_board_capabilities`
  - `ana_board_preview_message`
  - `ana_board_send_message`
  - `ana_board_current`
  - `ana_board_recent_messages`
  - `ana_board_clear`

Done when:

- Claude Code/OpenCode-style MCP clients can list tools
- agents can preview before sending
- send uses the same HTTP client as the CLI
- clear requires `confirm=true`

### Phase 11: Local SQLite

Goal:

Persist messages locally.

Build:

- schema
- migrations or simple init
- `SQLiteStore`
- message history persistence

Go concepts:

- `database/sql`
- drivers
- SQL rows
- scanning values
- resource cleanup

Done when:

- restart does not lose message history
- current frame can be restored

### Phase 12: API Token Auth

Goal:

Protect the API before public deployment.

Build:

- token middleware
- environment config
- protected POST endpoints
- public display route remains configurable

Go concepts:

- middleware
- closures
- environment variables
- constant-time compare later

Done when:

- missing token is rejected
- valid token is accepted
- admin and display behavior is intentional

### Phase 13: Cloudflare Container

Goal:

Deploy the Go server as a container.

Build:

- Dockerfile
- health endpoint
- container config
- Worker route to container

Go concepts:

- building static-ish binaries
- config via env
- graceful shutdown

Cloudflare concepts:

- Workers
- Containers
- wrangler
- routes

Done when:

- public URL serves board
- API can submit message
- display receives updates

### Phase 14: D1 Persistence

Goal:

Use Cloudflare D1 for production persistence.

Build:

- D1 schema
- Worker D1 binding
- internal Worker endpoints
- Go `D1BridgeStore` using HTTP

Schema v1:

```sql
CREATE TABLE messages (
  id TEXT PRIMARY KEY,
  text TEXT NOT NULL,
  source TEXT,
  priority TEXT NOT NULL DEFAULT 'normal',
  animation TEXT NOT NULL DEFAULT 'row',
  created_at INTEGER NOT NULL,
  displayed_at INTEGER,
  status TEXT NOT NULL DEFAULT 'displayed'
);

CREATE TABLE boards (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  rows INTEGER NOT NULL DEFAULT 6,
  cols INTEGER NOT NULL DEFAULT 22,
  current_frame_json TEXT,
  updated_at INTEGER NOT NULL
);

CREATE TABLE api_tokens (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  token_hash TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  revoked_at INTEGER
);
```

Done when:

- messages persist through container restart
- latest frame restores on boot
- D1 remains an implementation detail behind `Store`

### Phase 15: Agent Integrations

Goal:

Make Ana Board useful for real workflows.

Build:

- Codex example
- Claude Code MCP example
- OpenCode MCP example
- GitHub Actions example
- cron example
- Home Assistant or MQTT notes
- Hermes/Sense-style integration notes
- Tailscale private-network notes

Message examples:

```text
[blue]CODEX REVIEWING PR 42 💻
[red]BUILD FAILED AUTH TESTS ⚠️
[green]DEPLOY COMPLETE 🚀
[amber]AGENT NEEDS APPROVAL ⚠️
[violet]RESEARCH DIGEST READY 📌
```

Done when:

- docs show copy-pasteable integration examples
- integrations use the public API, not hidden internals

## 20. Go Concepts Mapped to Product Features

| Go concept | Product feature | Why it matters |
| --- | --- | --- |
| `struct` | Message, Frame, Cell | Names the important things in the system |
| function | Text normalization, layout | Turns input into output |
| method | Frame behavior | Keeps behavior attached to data |
| slice | Rows, cells, message history | Dynamic collections |
| map | Character set | Fast membership checks |
| error | Invalid text, invalid frame | Clear failure instead of hidden bugs |
| package | board, layout, server | Separates concerns |
| interface | Store | Lets storage change without rewriting app |
| context | HTTP lifecycle | Stops work when requests end |
| goroutine | SSE broadcaster | Lets work happen concurrently |
| channel | Frame updates | Sends events between parts |
| mutex | Memory store safety | Protects shared state |
| tests | Layout/store guarantees | Lets Codex change safely |
| embed | Static web files | Ships one binary |
| net/http | API and web UI | Standard Go web foundation |

## 21. Teaching Format Per Feature

Every implementation session should use this format:

```text
1. Plain-English system idea
2. The Go primitive that represents it
3. The files we will touch
4. The smallest code change
5. The verification command
6. What to inspect in the result
7. What not to add yet
```

Example:

```text
Feature:
Center text on the board.

System idea:
Turn arbitrary text into a fixed 6 x 22 visual frame.

Go primitive:
Pure function plus tests.

Files:
internal/layout/center.go
internal/layout/center_test.go

Verification:
go test ./internal/layout
```

## 22. Codex Collaboration Pattern

Because you want to learn the primitive puzzle pieces, Codex should not silently build large chunks.

For each phase, ask Codex for:

- what concept is being introduced
- what files will change
- why this is the smallest next step
- what command proves it works
- what you should read in the code afterward

Useful prompt template:

```text
Implement Phase N from docs/ana-board-spec.md.
Before editing, explain the Go primitive this phase teaches.
Keep the change small.
After editing, run the smallest relevant verification.
Then summarize what I should inspect to understand the concept.
```

Useful review prompt:

```text
Review the current implementation against Phase N of docs/ana-board-spec.md.
Focus on correctness, unnecessary complexity, and whether it still teaches the intended Go concept clearly.
```

## 23. Verification Strategy

Use the smallest relevant check first.

Early phases:

```text
go test ./internal/board
go test ./internal/layout
go test ./...
```

Server phases:

```text
go test ./...
go run ./cmd/ana-board
curl http://localhost:8080/healthz
curl http://localhost:8080/api/current
```

Browser phases:

```text
open http://localhost:8080
curl -X POST http://localhost:8080/api/messages ...
```

Cloudflare phases:

```text
wrangler deploy
curl https://<public-url>/healthz
```

Never claim a check passed unless it was actually run.

## 24. What We Are Avoiding in v0

Do not add these early:

- React
- Redux or frontend state libraries
- plugin system
- user accounts
- billing
- teams
- complex permissions
- distributed queue
- Kubernetes
- external production database
- OAuth
- multi-tenant architecture
- event sourcing
- observability stack

These may become useful later, but they are not the first problem.

## 25. Design Constraints

The visual design should be:

- focused on the board
- fullscreen-friendly
- calm and legible
- physically inspired, not gimmicky
- responsive enough for tablet and desktop

Avoid:

- stacked shadows
- double borders
- heavy dashboard chrome
- marketing hero sections
- decorative UI that competes with the board

The main screen should be the actual board.

## 26. Product Extensions

Possible future features:

- multiple boards
- per-source color accents
- message TTL
- schedule messages
- RSS feed source
- GitHub Actions integration
- Home Assistant integration
- MQTT support
- calendar summaries
- weather/status widgets
- exact frame API
- board themes
- public read-only display links
- kiosk mode
- PWA install
- Raspberry Pi deployment guide

Use this rule:

```text
An extension is allowed when the core message -> frame -> display loop is already solid.
```

## 27. Architecture Decision Records

Use lightweight ADRs when we make a meaningful choice.

Example location:

```text
docs/adr/0001-use-sse-before-websockets.md
```

Example format:

```text
# ADR 0001: Use SSE Before WebSockets

## Context
The display only needs one-way updates from the server.

## Decision
Use Server-Sent Events for v0 live updates.

## Consequences
Simpler server and client code. If we need bidirectional sessions later, we can add WebSockets.
```

## 28. Open Questions

These should be answered as the product takes shape:

1. Is the final name Ana Board, AgentBoard, or something else?
2. Should the default board exactly match 22 x 6, or allow configurable dimensions from day one?
3. Should unsupported characters become spaces or `?`?
4. Should messages replace the board immediately, or should v1 include a queue?
5. Should the display route be public by default when deployed?
6. Should local persistence be SQLite before Cloudflare D1?
7. Should the browser animation simulate actual split flaps or simply flip characters?

Default answers for v0:

1. Use Ana Board as the repo/product name for now.
2. Hard-code 22 x 6.
3. Use `?` for visible unsupported characters, space for whitespace.
4. Replace immediately.
5. Local public display is fine; deployed write API needs a token.
6. Memory first, SQLite later, D1 after Cloudflare deployment.
7. Start simple, improve animation after correctness.

## 29. Definition of v0

v0 is complete when:

- Go server runs locally
- browser displays a 22 x 6 board
- API accepts messages
- text is normalized and centered
- current frame is available via API
- browser updates live through SSE
- basic split-flap animation works
- recent messages are visible in admin or API
- README explains local run and curl usage
- tests cover board and layout behavior

v0 is not required to include:

- Cloudflare
- D1
- auth
- SQLite
- CLI
- advanced queue
- multiple boards
- polished integrations

## 30. Immediate Next Step

Start with Phase 0 and Phase 1.

The first implementation should create:

```text
go.mod
cmd/ana-board/main.go
internal/board/cell.go
internal/board/frame.go
internal/board/frame_test.go
README.md
```

The first working command:

```text
go test ./...
```

The first learning goal:

Understand that a Go program is built from small named packages, and that a `Frame` is just a precise data shape with rules that prevent invalid board states.

## 31. First Five Learning Sessions

These are the recommended first five working sessions.

### Session 1: What Is a Go Program?

Build:

- `go.mod`
- `cmd/ana-board/main.go`
- a basic `main` function

Learn:

- what a module is
- what a package is
- why `main` is special
- how `go run` works
- how `go test` discovers tests

Architect takeaway:

```text
A Go app is a module containing packages. One package named main produces the executable.
```

### Session 2: What Is a Board Frame?

Build:

- `Cell`
- `Frame`
- `NewFrame`
- tests for frame dimensions

Learn:

- structs
- constructors
- slices
- invariants

Architect takeaway:

```text
Good software starts by naming the real things in the problem.
For Ana Board, the first real thing is a fixed-size Frame.
```

### Session 3: How Does Text Become Safe?

Build:

- character set
- text normalization
- validation errors

Learn:

- strings vs runes
- maps
- loops
- errors

Architect takeaway:

```text
Input from the outside world is messy. Boundaries clean it before it reaches the core system.
```

### Session 4: How Does Text Become Layout?

Build:

- centered layout
- word wrapping
- too-large message errors

Learn:

- pure functions
- table-driven tests
- behavior-first verification

Architect takeaway:

```text
The layout engine is a transformation: text in, frame out. That makes it easy to test.
```

### Session 5: How Does the Outside World Talk to It?

Build:

- `GET /healthz`
- `POST /api/messages`
- `GET /api/current`

Learn:

- HTTP handlers
- JSON
- status codes
- request validation

Architect takeaway:

```text
The server is an adapter. It turns network requests into calls into the core engine.
```

After these five sessions, you will understand the core shape of the whole project.

## 32. Phase Summary Table

| Phase | Build result | Main Go concept | Verification |
| --- | --- | --- | --- |
| 0 | project skeleton | modules and packages | `go test ./...` |
| 1 | board frame | structs and slices | board tests |
| 2 | safe text | strings, runes, errors | normalization tests |
| 3 | centered layout | pure functions | layout tests |
| 4 | local API | `net/http`, JSON | curl API checks |
| 5 | memory store | interfaces | store tests |
| 6 | browser board | static files, templates | open localhost |
| 7 | live updates | goroutines, channels | SSE browser update |
| 8 | animation | state vs presentation | visual check |
| 9 | admin panel | form handling | browser submit |
| 10 | CLI | args, HTTP client | `ana-board send` |
| 11 | SQLite | `database/sql` | restart persistence |
| 12 | auth | middleware | token checks |
| 13 | Cloudflare container | deployable binary | public health check |
| 14 | D1 | storage boundary | restart recovery |
| 15 | integrations | public API design | Codex/GitHub examples |

## 33. How To Know We Are Going Too Fast

Slow down if any of these happen:

- a phase changes more than four or five concepts at once
- we add a dependency before understanding the standard-library version
- tests become hard to explain
- `main.go` starts collecting business logic
- a feature requires Cloudflare before it works locally
- the UI becomes a dashboard before the board works
- storage details leak into layout or server code
- the code works, but you cannot explain the data flow in one minute

The corrective move is always:

```text
Return to the primitive.
Name the data.
Name the transformation.
Write the smallest test.
```
