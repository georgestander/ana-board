package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/georgestander/ana-board/internal/art"
	"github.com/georgestander/ana-board/internal/board"
)

func TestParseMessageCommandSupportsExactPlacements(t *testing.T) {
	parsed, err := parseMessageCommand("frame", []string{
		"--placements-json", `[{"row":0,"col":0,"symbol":"A","color":"green"}]`,
		"--source", "codex",
	})
	if err != nil {
		t.Fatalf("parseMessageCommand returned error: %v", err)
	}

	if len(parsed.Request.Placements) != 1 {
		t.Fatalf("placements = %d, want 1", len(parsed.Request.Placements))
	}
	if parsed.Request.Placements[0].Row != 0 || parsed.Request.Placements[0].Col != 0 {
		t.Fatalf("placement coordinate = %d/%d, want 0/0", parsed.Request.Placements[0].Row, parsed.Request.Placements[0].Col)
	}
}

func TestParseMessageCommandRejectsMixedPayloads(t *testing.T) {
	_, err := parseMessageCommand("send", []string{
		"--placements-json", `[{"row":0,"col":0,"symbol":"A"}]`,
		"hello",
	})
	if err == nil {
		t.Fatal("parseMessageCommand returned nil error, want mixed payload error")
	}
}

func TestParseMessageCommandSupportsSprite(t *testing.T) {
	parsed, err := parseMessageCommand("frame", []string{
		"--sprite", "trophy",
		"--source", "codex",
	})
	if err != nil {
		t.Fatalf("parseMessageCommand returned error: %v", err)
	}

	if parsed.Request.Frame == nil {
		t.Fatal("frame = nil, want sprite frame")
	}
	if parsed.Request.Frame.Cells[0][0] != " " {
		t.Fatalf("first cell = %q, want blank", parsed.Request.Frame.Cells[0][0])
	}
}

func TestParseMessageCommandSupportsImage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tiny.png")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	img := image.NewNRGBA(image.Rect(0, 0, board.DefaultCols, board.DefaultRows))
	img.SetNRGBA(0, 0, color.NRGBA{R: 228, G: 191, B: 114, A: 255})
	if err := png.Encode(file, img); err != nil {
		t.Fatalf("png.Encode returned error: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	parsed, err := parseMessageCommand("frame", []string{
		"--image", path,
		"--source", "codex",
	})
	if err != nil {
		t.Fatalf("parseMessageCommand returned error: %v", err)
	}

	if parsed.Request.Frame == nil {
		t.Fatal("frame = nil, want image frame")
	}
	if parsed.Request.Frame.Cells[0][0] != art.PixelSymbol {
		t.Fatalf("first cell = %q, want block pixel", parsed.Request.Frame.Cells[0][0])
	}
}

func TestParseExactTimeSupportsRFC3339(t *testing.T) {
	now := time.Date(2026, 5, 31, 18, 0, 0, 0, time.FixedZone("SAST", 2*60*60))
	got, err := parseExactTime("2026-05-31T18:30:00+02:00", now)
	if err != nil {
		t.Fatalf("parseExactTime returned error: %v", err)
	}

	if got.Format(time.RFC3339) != "2026-05-31T18:30:00+02:00" {
		t.Fatalf("time = %s, want 2026-05-31T18:30:00+02:00", got.Format(time.RFC3339))
	}
}

func TestParseExactTimeRejectsPastTime(t *testing.T) {
	now := time.Date(2026, 5, 31, 18, 0, 0, 0, time.UTC)
	_, err := parseExactTime("2026-05-31T17:59:00Z", now)
	if err == nil {
		t.Fatal("parseExactTime returned nil error, want past time error")
	}
}
