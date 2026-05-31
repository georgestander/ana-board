package board

import "testing"

func TestNewFrameCreatesBlankFrame(t *testing.T) {
	frame, err := NewFrame(DefaultRows, DefaultCols)
	if err != nil {
		t.Fatalf("NewFrame returned error: %v", err)
	}

	if frame.Rows != DefaultRows {
		t.Fatalf("Rows = %d, want %d", frame.Rows, DefaultRows)
	}

	if frame.Cols != DefaultCols {
		t.Fatalf("Cols = %d, want %d", frame.Cols, DefaultCols)
	}

	if len(frame.Cells) != DefaultRows {
		t.Fatalf("len(Cells) = %d, want %d", len(frame.Cells), DefaultRows)
	}

	if len(frame.Cells[0]) != DefaultCols {
		t.Fatalf("len(Cells[0]) = %d, want %d", len(frame.Cells[0]), DefaultCols)
	}

	if frame.Cells[0][0].Symbol != " " {
		t.Fatalf("first cell = %q, want space", frame.Cells[0][0].Symbol)
	}
}

func TestNewFrameRejectsInvalidDimensions(t *testing.T) {
	tests := []struct {
		name string
		rows int
		cols int
	}{
		{
			name: "zero rows",
			rows: 0,
			cols: DefaultCols,
		},
		{
			name: "zero cols",
			rows: DefaultRows,
			cols: 0,
		},
		{
			name: "negative rows",
			rows: -1,
			cols: DefaultCols,
		},
		{
			name: "negative cols",
			rows: DefaultRows,
			cols: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewFrame(tt.rows, tt.cols)
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestFrameSetUpdatesCell(t *testing.T) {
	frame, err := NewFrame(DefaultRows, DefaultCols)
	if err != nil {
		t.Fatalf("NewFrame returned error: %v", err)
	}

	if err := frame.Set(1, 2, Cell{Symbol: "A"}); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}

	cell, err := frame.CellAt(1, 2)
	if err != nil {
		t.Fatalf("CellAt returned error: %v", err)
	}

	if cell.Symbol != "A" {
		t.Fatalf("cell = %q, want %q", cell.Symbol, "A")
	}
}

func TestFrameSetRejectsInvalidCoordinates(t *testing.T) {
	frame, err := NewFrame(DefaultRows, DefaultCols)
	if err != nil {
		t.Fatalf("NewFrame returned error: %v", err)
	}

	tests := []struct {
		name string
		row  int
		col  int
	}{
		{
			name: "negative row",
			row:  -1,
			col:  0,
		},
		{
			name: "row too high",
			row:  DefaultRows,
			col:  0,
		},
		{
			name: "negative col",
			row:  0,
			col:  -1,
		},
		{
			name: "col too high",
			row:  0,
			col:  DefaultCols,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := frame.Set(tt.row, tt.col, Cell{Symbol: "A"}); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestFrameCellAtReturnsCell(t *testing.T) {
	frame, err := NewFrame(DefaultRows, DefaultCols)
	if err != nil {
		t.Fatalf("NewFrame returned error: %v", err)
	}

	if err := frame.Set(3, 4, Cell{Symbol: "Z"}); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}

	cell, err := frame.CellAt(3, 4)
	if err != nil {
		t.Fatalf("CellAt returned error: %v", err)
	}

	if cell.Symbol != "Z" {
		t.Fatalf("cell = %q, want %q", cell.Symbol, "Z")
	}
}

func TestFrameCellAtRejectsInvalidCoordinates(t *testing.T) {
	frame, err := NewFrame(DefaultRows, DefaultCols)
	if err != nil {
		t.Fatalf("NewFrame returned error: %v", err)
	}

	tests := []struct {
		name string
		row  int
		col  int
	}{
		{
			name: "negative row",
			row:  -1,
			col:  0,
		},
		{
			name: "row too high",
			row:  DefaultRows,
			col:  0,
		},
		{
			name: "negative col",
			row:  0,
			col:  -1,
		},
		{
			name: "col too high",
			row:  0,
			col:  DefaultCols,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := frame.CellAt(tt.row, tt.col); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
