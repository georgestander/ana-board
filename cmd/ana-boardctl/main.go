package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/georgestander/ana-board/internal/art"
	"github.com/georgestander/ana-board/internal/board"
	"github.com/georgestander/ana-board/internal/capabilities"
	"github.com/georgestander/ana-board/internal/client"
	"github.com/georgestander/ana-board/internal/layout"
	"github.com/georgestander/ana-board/internal/messages"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printHelp()
		return nil
	}

	switch args[0] {
	case "art":
		return runArt(args[1:])
	case "capabilities":
		return runCapabilities(args[1:])
	case "preview":
		return runPreview(args[1:])
	case "send":
		return runSend(args[1:])
	case "frame":
		return runFrame(args[1:])
	case "current":
		return runCurrent(args[1:])
	case "recent":
		return runRecent(args[1:])
	case "clear":
		return runClear(args[1:])
	case "help", "-h", "--help":
		printHelp()
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runArt(args []string) error {
	if len(args) == 0 || args[0] == "list" {
		for _, name := range art.ListSprites() {
			fmt.Println(name)
		}
		return nil
	}

	return fmt.Errorf("unknown art command %q", args[0])
}

func runCapabilities(args []string) error {
	fs := flag.NewFlagSet("capabilities", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "print JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}

	caps := capabilities.Current()
	if *jsonOut {
		return printJSON(caps)
	}

	fmt.Printf("Board: %d rows x %d cols (%d tiles)\n", caps.Board.Rows, caps.Board.Cols, caps.Board.MaxTiles)
	fmt.Printf("Colors: %s\n", strings.Join(caps.Colors, ", "))
	fmt.Printf("Kinds: %s\n", strings.Join(caps.Kinds, ", "))
	fmt.Printf("Animations: %s\n", strings.Join(caps.Animations, ", "))
	fmt.Printf("Emoji: %s\n", caps.Text.EmojiSupport)
	fmt.Printf("Color syntax: %s\n", caps.Text.ColorSyntax)
	return nil
}

func runPreview(args []string) error {
	parsed, err := parseMessageCommand("preview", args)
	if err != nil {
		return err
	}

	frame, err := requestFrame(parsed.Request)
	if err != nil {
		return err
	}

	out := frameToOutput(frame)
	if parsed.JSONOut {
		return printJSON(out)
	}

	for _, row := range out.Cells {
		fmt.Println(strings.Join(row, ""))
	}
	return nil
}

func requestFrame(req messages.SubmitRequest) (board.Frame, error) {
	if len(req.Placements) != 0 {
		frame, _, _, err := layout.ExactFrameFromPlacements(req.Placements, req.Color)
		return frame, err
	}

	if req.Frame != nil {
		frame, _, _, err := layout.ExactFrameFromInput(*req.Frame, req.Color)
		return frame, err
	}

	cells, err := requestCells(req)
	if err != nil {
		return board.Frame{}, err
	}

	return layout.CenterCells(cells)
}

func requestCells(req messages.SubmitRequest) ([]board.Cell, error) {
	if len(req.Tiles) != 0 {
		cells := make([]board.Cell, 0, len(req.Tiles))
		for _, tile := range req.Tiles {
			color, err := messages.NormalizeColor(tile.Color)
			if err != nil {
				return nil, err
			}
			if tile.Color == "" {
				color = req.Color
			}

			cell, err := layout.NormalizeTileCell(tile.Symbol, color)
			if err != nil {
				return nil, err
			}
			cells = append(cells, cell)
		}

		return cells, nil
	}

	if len(req.Segments) == 0 {
		return board.NormalizeCells(req.Text, req.Color)
	}

	segments := make([]board.TextSegment, len(req.Segments))
	for index, segment := range req.Segments {
		segments[index] = board.TextSegment{Text: segment.Text, Color: segment.Color}
	}

	return board.NormalizeSegmentCells(segments, req.Color)
}

func runSend(args []string) error {
	parsed, err := parseMessageCommand("send", args)
	if err != nil {
		return err
	}

	if err := waitUntil(parsed.At); err != nil {
		return err
	}

	boardClient, err := client.New(parsed.BaseURL)
	if err != nil {
		return err
	}

	resp, err := boardClient.SendMessage(context.Background(), parsed.Request)
	if err != nil {
		return err
	}

	if parsed.JSONOut {
		return printJSON(resp)
	}

	fmt.Printf("sent %s from %s\n", resp.ID, resp.Message.Source)
	return nil
}

func runFrame(args []string) error {
	parsed, err := parseMessageCommand("frame", args)
	if err != nil {
		return err
	}
	if len(parsed.Request.Placements) == 0 && parsed.Request.Frame == nil {
		return fmt.Errorf("frame requires --placements-json or --frame-json")
	}

	if err := waitUntil(parsed.At); err != nil {
		return err
	}

	boardClient, err := client.New(parsed.BaseURL)
	if err != nil {
		return err
	}

	resp, err := boardClient.SendMessage(context.Background(), parsed.Request)
	if err != nil {
		return err
	}

	if parsed.JSONOut {
		return printJSON(resp)
	}

	fmt.Printf("sent frame %s from %s\n", resp.ID, resp.Message.Source)
	return nil
}

func runCurrent(args []string) error {
	fs := flag.NewFlagSet("current", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "print JSON")
	baseURL := fs.String("url", defaultBaseURL(), "board URL")
	if err := fs.Parse(args); err != nil {
		return err
	}

	boardClient, err := client.New(*baseURL)
	if err != nil {
		return err
	}

	resp, err := boardClient.CurrentFrame(context.Background())
	if err != nil {
		return err
	}

	if *jsonOut {
		return printJSON(resp)
	}

	for _, row := range resp.Frame.Cells {
		fmt.Println(strings.Join(row, ""))
	}
	return nil
}

func runRecent(args []string) error {
	fs := flag.NewFlagSet("recent", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "print JSON")
	baseURL := fs.String("url", defaultBaseURL(), "board URL")
	limit := fs.Int("limit", 10, "message limit")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *limit <= 0 {
		return fmt.Errorf("limit must be a positive integer")
	}

	boardClient, err := client.New(*baseURL)
	if err != nil {
		return err
	}

	resp, err := boardClient.ListMessages(context.Background(), *limit)
	if err != nil {
		return err
	}

	if *jsonOut {
		return printJSON(resp)
	}

	for _, msg := range resp.Messages {
		fmt.Printf("%s [%s/%s/%s] %s\n", msg.CreatedAt.Format("2006-01-02 15:04:05"), msg.Source, msg.Kind, msg.Color, msg.Text)
	}
	return nil
}

func runClear(args []string) error {
	fs := flag.NewFlagSet("clear", flag.ContinueOnError)
	jsonOut := fs.Bool("json", false, "print JSON")
	baseURL := fs.String("url", defaultBaseURL(), "board URL")
	confirm := fs.Bool("confirm", false, "confirm clearing the board")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if !*confirm {
		return fmt.Errorf("clear requires --confirm")
	}

	boardClient, err := client.New(*baseURL)
	if err != nil {
		return err
	}

	resp, err := boardClient.Clear(context.Background())
	if err != nil {
		return err
	}

	if *jsonOut {
		return printJSON(resp)
	}

	fmt.Println("board cleared")
	return nil
}

type parsedMessageCommand struct {
	Request messages.SubmitRequest
	JSONOut bool
	BaseURL string
	At      string
}

func parseMessageCommand(name string, args []string) (parsedMessageCommand, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	animation := fs.String("animation", messages.DefaultAnimation, "animation: row")
	color := fs.String("color", messages.DefaultColor, "default color for untagged tiles")
	kind := fs.String("kind", messages.DefaultKind, "kind")
	priority := fs.String("priority", messages.DefaultPriority, "priority")
	source := fs.String("source", "ana-boardctl", "source")
	baseURL := fs.String("url", defaultBaseURL(), "board URL")
	segmentsJSON := fs.String("segments-json", "", "JSON array of colored text segments")
	tilesJSON := fs.String("tiles-json", "", "JSON array of exact per-tile symbols and colors")
	placementsJSON := fs.String("placements-json", "", "JSON array of row/col tile placements")
	frameJSON := fs.String("frame-json", "", "JSON object with 6x22 cells and optional colors")
	sprite := fs.String("sprite", "", "named block-art sprite")
	imagePath := fs.String("image", "", "image file to convert to block art")
	at := fs.String("at", "", "optional exact send time, RFC3339 or local 'YYYY-MM-DD HH:MM[:SS]'")
	jsonOut := fs.Bool("json", false, "print JSON")
	if err := fs.Parse(args); err != nil {
		return parsedMessageCommand{}, err
	}

	text := strings.TrimSpace(strings.Join(fs.Args(), " "))
	hasSegments := strings.TrimSpace(*segmentsJSON) != ""
	hasTiles := strings.TrimSpace(*tilesJSON) != ""
	hasPlacements := strings.TrimSpace(*placementsJSON) != ""
	hasFrame := strings.TrimSpace(*frameJSON) != ""
	hasSprite := strings.TrimSpace(*sprite) != ""
	hasImage := strings.TrimSpace(*imagePath) != ""
	payloadCount := 0
	for _, hasPayload := range []bool{text != "", hasSegments, hasTiles, hasPlacements, hasFrame, hasSprite, hasImage} {
		if hasPayload {
			payloadCount++
		}
	}
	if payloadCount == 0 {
		return parsedMessageCommand{}, fmt.Errorf("%s requires message text, --segments-json, --tiles-json, --placements-json, --frame-json, --sprite, or --image", name)
	}
	if payloadCount > 1 {
		return parsedMessageCommand{}, fmt.Errorf("use either text, --segments-json, --tiles-json, --placements-json, --frame-json, --sprite, or --image")
	}

	req := messages.SubmitRequest{
		Text:      text,
		Source:    *source,
		Priority:  *priority,
		Animation: *animation,
		Kind:      *kind,
		Color:     *color,
	}
	if hasSegments {
		if err := json.Unmarshal([]byte(*segmentsJSON), &req.Segments); err != nil {
			return parsedMessageCommand{}, fmt.Errorf("invalid --segments-json: %w", err)
		}
	}
	if hasTiles {
		if err := json.Unmarshal([]byte(*tilesJSON), &req.Tiles); err != nil {
			return parsedMessageCommand{}, fmt.Errorf("invalid --tiles-json: %w", err)
		}
	}
	if hasPlacements {
		if err := json.Unmarshal([]byte(*placementsJSON), &req.Placements); err != nil {
			return parsedMessageCommand{}, fmt.Errorf("invalid --placements-json: %w", err)
		}
	}
	if hasFrame {
		var frame messages.FrameInput
		if err := json.Unmarshal([]byte(*frameJSON), &frame); err != nil {
			return parsedMessageCommand{}, fmt.Errorf("invalid --frame-json: %w", err)
		}
		req.Frame = &frame
	}
	if hasSprite {
		frame, err := art.SpriteFrame(*sprite)
		if err != nil {
			return parsedMessageCommand{}, err
		}
		req.Frame = &frame
	}
	if hasImage {
		file, err := os.Open(*imagePath)
		if err != nil {
			return parsedMessageCommand{}, err
		}
		defer file.Close()

		frame, err := art.ImageFrame(file)
		if err != nil {
			return parsedMessageCommand{}, err
		}
		req.Frame = &frame
	}

	var err error
	req.Source, err = messages.NormalizeSource(req.Source)
	if err != nil {
		return parsedMessageCommand{}, err
	}
	req.Priority, err = messages.NormalizePriority(req.Priority)
	if err != nil {
		return parsedMessageCommand{}, err
	}
	req.Animation, err = messages.NormalizeAnimation(req.Animation)
	if err != nil {
		return parsedMessageCommand{}, err
	}
	req.Kind, err = messages.NormalizeKind(req.Kind)
	if err != nil {
		return parsedMessageCommand{}, err
	}
	req.Color, err = messages.NormalizeColor(req.Color)
	if err != nil {
		return parsedMessageCommand{}, err
	}

	return parsedMessageCommand{
		Request: req,
		JSONOut: *jsonOut,
		BaseURL: *baseURL,
		At:      strings.TrimSpace(*at),
	}, nil
}

func waitUntil(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	target, err := parseExactTime(raw, time.Now())
	if err != nil {
		return err
	}

	delay := time.Until(target)
	if delay <= 0 {
		return fmt.Errorf("--at time must be in the future")
	}

	fmt.Fprintf(os.Stderr, "waiting until %s\n", target.Format(time.RFC3339))
	timer := time.NewTimer(delay)
	defer timer.Stop()
	<-timer.C
	return nil
}

func parseExactTime(raw string, now time.Time) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, fmt.Errorf("--at time is required")
	}

	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		parsed, err := time.Parse(layout, raw)
		if err == nil {
			if !parsed.After(now) {
				return time.Time{}, fmt.Errorf("--at time must be in the future")
			}
			return parsed, nil
		}
	}

	for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02 15:04"} {
		parsed, err := time.ParseInLocation(layout, raw, time.Local)
		if err == nil {
			if !parsed.After(now) {
				return time.Time{}, fmt.Errorf("--at time must be in the future")
			}
			return parsed, nil
		}
	}

	return time.Time{}, fmt.Errorf("--at must be RFC3339 or local time like 2026-05-31 18:30:00")
}

func frameToOutput(frame board.Frame) client.FrameResponse {
	cells := make([][]string, frame.Rows)
	colors := make([][]string, frame.Rows)
	for row := range cells {
		cells[row] = make([]string, frame.Cols)
		colors[row] = make([]string, frame.Cols)
		for col := range cells[row] {
			cell, err := frame.CellAt(row, col)
			if err != nil {
				cells[row][col] = " "
				colors[row][col] = board.DefaultColor
				continue
			}
			cells[row][col] = cell.Symbol
			colors[row][col] = cell.Color
		}
	}

	return client.FrameResponse{
		Rows:   frame.Rows,
		Cols:   frame.Cols,
		Cells:  cells,
		Colors: colors,
	}
}

func defaultBaseURL() string {
	value := os.Getenv("ANA_BOARD_URL")
	if strings.TrimSpace(value) == "" {
		return client.DefaultBaseURL
	}

	return value
}

func printJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func printHelp() {
	fmt.Println(`ana-boardctl sends concise updates to Ana Board.

Commands:
  art list
  capabilities [--json]
  preview "[blue]HELLO 🌍"
  send [--url URL] [--source SOURCE] [--kind KIND] "[green]BUILD PASSED ✅"
  send --tiles-json '[{"symbol":"A","color":"green"},{"symbol":"N","color":"amber"},{"symbol":"A","color":"red"}]'
  send --segments-json '[{"text":"OK ","color":"green"},{"text":"FAIL","color":"red"}]'
  send --sprite trophy --source codex
  frame --image ./tiny.png --source codex
  frame --placements-json '[{"row":0,"col":0,"symbol":"A","color":"green"}]'
  frame --frame-json '{"cells":[["A",... 22 columns],... 6 rows]}'
  send --at "2026-05-31T18:30:00+02:00" "[amber]REMINDER ⏰"
  current [--url URL] [--json]
  recent [--url URL] [--limit N] [--json]
  clear [--url URL] --confirm

Environment:
  ANA_BOARD_URL defaults to http://localhost:8080

Notes:
  Native iOS/macOS emoji can be pasted directly; aliases are optional shortcuts.
  Only row animation is supported.
  --color is only the default. Use --tiles-json for exact per-letter color, or [green] inline tokens for quick text.
  --sprite and --image produce exact frames made from colored █ block pixels.
  Use --placements-json or --frame-json when the agent needs exact row/column control.
  --at waits client-side, then sends at that exact time.`)
}
