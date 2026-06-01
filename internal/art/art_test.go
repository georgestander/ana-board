package art

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/georgestander/ana-board/internal/board"
)

func TestListSpritesIncludesExpectedNames(t *testing.T) {
	names := ListSprites()
	for _, want := range []string{"check", "heart", "rocket", "trophy", "warning", "wrench", "x"} {
		if !contains(names, want) {
			t.Fatalf("sprites = %v, want %q", names, want)
		}
	}
}

func TestSpriteFrameProducesBoardSizedBlockArt(t *testing.T) {
	frame, err := SpriteFrame("check")
	if err != nil {
		t.Fatalf("SpriteFrame returned error: %v", err)
	}

	if len(frame.Cells) != board.DefaultRows {
		t.Fatalf("rows = %d, want %d", len(frame.Cells), board.DefaultRows)
	}
	if len(frame.Cells[0]) != board.DefaultCols {
		t.Fatalf("cols = %d, want %d", len(frame.Cells[0]), board.DefaultCols)
	}

	foundBlock := false
	for row := range frame.Cells {
		for col := range frame.Cells[row] {
			if frame.Cells[row][col] == PixelSymbol {
				foundBlock = true
				if frame.Colors[row][col] != "green" {
					t.Fatalf("block color = %q, want green", frame.Colors[row][col])
				}
			}
		}
	}
	if !foundBlock {
		t.Fatal("sprite did not produce any block pixels")
	}
}

func TestSpriteFrameRejectsUnknownName(t *testing.T) {
	_, err := SpriteFrame("nope")
	if err == nil {
		t.Fatal("SpriteFrame returned nil error, want unknown sprite error")
	}
}

func TestImageFrameMapsOpaquePixelsToNearestPaletteColor(t *testing.T) {
	var buf bytes.Buffer
	img := image.NewNRGBA(image.Rect(0, 0, 22, 6))
	for y := 0; y < 6; y++ {
		for x := 0; x < 22; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 148, G: 213, B: 155, A: 255})
		}
	}
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode returned error: %v", err)
	}

	frame, err := ImageFrame(&buf)
	if err != nil {
		t.Fatalf("ImageFrame returned error: %v", err)
	}

	if frame.Cells[0][0] != PixelSymbol {
		t.Fatalf("cell = %q, want block pixel", frame.Cells[0][0])
	}
	if frame.Colors[0][0] != "green" {
		t.Fatalf("color = %q, want green", frame.Colors[0][0])
	}
}

func TestImageFrameTreatsTransparentPixelsAsBlank(t *testing.T) {
	var buf bytes.Buffer
	img := image.NewNRGBA(image.Rect(0, 0, 22, 6))
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode returned error: %v", err)
	}

	frame, err := ImageFrame(&buf)
	if err != nil {
		t.Fatalf("ImageFrame returned error: %v", err)
	}

	if frame.Cells[0][0] != " " {
		t.Fatalf("cell = %q, want blank", frame.Cells[0][0])
	}
	if frame.Colors[0][0] != board.DefaultColor {
		t.Fatalf("color = %q, want default", frame.Colors[0][0])
	}
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
