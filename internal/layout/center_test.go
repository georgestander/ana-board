package layout

import (
	"testing"

	"github.com/georgestander/ana-board/internal/board"
)

func TestCenterTextCentersShortMessage(t *testing.T) {
	frame, err := CenterText("hello")
	if err != nil {
		t.Fatalf("CenterText returned error: %v", err)
	}

	tests := []struct {
		name string
		row  int
		col  int
		want string
	}{
		{
			name: "first letter",
			row:  2,
			col:  8,
			want: "H",
		},
		{
			name: "last letter",
			row:  2,
			col:  12,
			want: "O",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cell, err := frame.CellAt(tt.row, tt.col)
			if err != nil {
				t.Fatalf("CellAt returned error: %v", err)
			}

			if cell.Symbol != tt.want {
				t.Fatalf("cell = %q, want %q", cell.Symbol, tt.want)
			}
		})
	}
}

func TestCenterTextRejectsWordLongerThanOneRow(t *testing.T) {
	_, err := CenterText("supercalifragilisticexpialidocious")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWrapWordsKeepsShortTextOnOneLine(t *testing.T) {
	lines, err := wrapWords("HELLO ANA")
	if err != nil {
		t.Fatalf("wrapWords returned error: %v", err)
	}

	if len(lines) != 1 {
		t.Fatalf("len(lines) = %d, want 1", len(lines))
	}

	if lines[0] != "HELLO ANA" {
		t.Fatalf("lines[0] = %q, want %q", lines[0], "HELLO ANA")
	}
}

func TestWrapWordsSplitsTextAcrossLines(t *testing.T) {
	lines, err := wrapWords("HELLO FROM ANA BOARD STATUS READY")
	if err != nil {
		t.Fatalf("wrapWords returned error: %v", err)
	}

	if len(lines) != 2 {
		t.Fatalf("len(lines) = %d, want 2", len(lines))
	}

	if lines[0] != "HELLO FROM ANA BOARD" {
		t.Fatalf("lines[0] = %q, want %q", lines[0], "HELLO FROM ANA BOARD")
	}

	if lines[1] != "STATUS READY" {
		t.Fatalf("lines[1] = %q, want %q", lines[1], "STATUS READY")
	}
}

func TestCenterTextCentersWrappedMessage(t *testing.T) {
	frame, err := CenterText("HELLO FROM ANA BOARD STATUS READY")
	if err != nil {
		t.Fatalf("CenterText returned error: %v", err)
	}

	tests := []struct {
		name string
		row  int
		col  int
		want string
	}{
		{
			name: "first line starts centered",
			row:  2,
			col:  1,
			want: "H",
		},
		{
			name: "second line starts centered",
			row:  3,
			col:  5,
			want: "S",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cell, err := frame.CellAt(tt.row, tt.col)
			if err != nil {
				t.Fatalf("CellAt returned error: %v", err)
			}

			if cell.Symbol != tt.want {
				t.Fatalf("cell = %q, want %q", cell.Symbol, tt.want)
			}
		})
	}
}

func TestCenterTextPlacesNativeEmojiAsOneTileWithPerTileColor(t *testing.T) {
	frame, err := CenterTextWithColor("[green]hello [blue]🌍", "white")
	if err != nil {
		t.Fatalf("CenterTextWithColor returned error: %v", err)
	}

	first, err := frame.CellAt(2, 7)
	if err != nil {
		t.Fatalf("CellAt returned error: %v", err)
	}
	if first.Symbol != "H" || first.Color != "green" {
		t.Fatalf("first cell = %#v, want green H", first)
	}

	cell, err := frame.CellAt(2, 13)
	if err != nil {
		t.Fatalf("CellAt returned error: %v", err)
	}

	if cell.Symbol != "🌍" {
		t.Fatalf("cell = %q, want globe emoji", cell.Symbol)
	}

	if cell.Color != "blue" {
		t.Fatalf("color = %q, want blue", cell.Color)
	}
}

func TestCenterSegmentsAppliesPerTileColors(t *testing.T) {
	frame, err := CenterSegments([]board.TextSegment{
		{Text: "ANA ", Color: "green"},
		{Text: "READY ✅", Color: "blue"},
	}, "white")
	if err != nil {
		t.Fatalf("CenterSegments returned error: %v", err)
	}

	first, err := frame.CellAt(2, 5)
	if err != nil {
		t.Fatalf("CellAt returned error: %v", err)
	}
	if first.Symbol != "A" || first.Color != "green" {
		t.Fatalf("first = %#v, want green A", first)
	}

	emoji, err := frame.CellAt(2, 15)
	if err != nil {
		t.Fatalf("CellAt returned error: %v", err)
	}
	if emoji.Symbol != "✅" || emoji.Color != "blue" {
		t.Fatalf("emoji = %#v, want blue check", emoji)
	}
}

func TestCenterTextRejectsTooManyLines(t *testing.T) {
	_, err := CenterText("ABCDEFGHIJKLMNOPQRSTUV ABCDEFGHIJKLMNOPQRSTUV ABCDEFGHIJKLMNOPQRSTUV ABCDEFGHIJKLMNOPQRSTUV ABCDEFGHIJKLMNOPQRSTUV ABCDEFGHIJKLMNOPQRSTUV ABCDEFGHIJKLMNOPQRSTUV")
	if err == nil {
		t.Fatal("expected error")
	}
}
