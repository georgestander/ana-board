package board

import "fmt"

const (
	DefaultRows  = 6
	DefaultCols  = 22
	DefaultColor = "white"
)

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
			cells[row][col] = NewCell(" ", DefaultColor)
		}
	}

	return Frame{
		Rows:  rows,
		Cols:  cols,
		Cells: cells,
	}, nil
}

func (f *Frame) Set(row, col int, cell Cell) error {
	if err := f.validateCoordinates(row, col); err != nil {
		return err
	}

	cell = NewCell(cell.Symbol, cell.Color)
	f.Cells[row][col] = cell
	return nil
}

func (f Frame) CellAt(row, col int) (Cell, error) {
	if err := f.validateCoordinates(row, col); err != nil {
		return Cell{}, err
	}

	return f.Cells[row][col], nil
}

func (f Frame) validateCoordinates(row, col int) error {
	if row < 0 || row >= f.Rows {
		return fmt.Errorf("row %d is outside frame", row)
	}

	if col < 0 || col >= f.Cols {
		return fmt.Errorf("col %d is outside frame", col)
	}

	return nil
}
