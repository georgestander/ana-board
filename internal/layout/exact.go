package layout

import (
	"fmt"
	"strings"

	"github.com/georgestander/ana-board/internal/board"
	"github.com/georgestander/ana-board/internal/messages"
)

func ExactFrameFromPlacements(placements []messages.PlacedTile, defaultColor string) (board.Frame, []messages.PlacedTile, string, error) {
	frame, err := board.NewFrame(board.DefaultRows, board.DefaultCols)
	if err != nil {
		return board.Frame{}, nil, "", err
	}
	if len(placements) == 0 {
		return board.Frame{}, nil, "", fmt.Errorf("placements are required")
	}

	defaultColor, err = messages.NormalizeColor(defaultColor)
	if err != nil {
		return board.Frame{}, nil, "", err
	}

	seen := make(map[[2]int]bool, len(placements))
	stored := make([]messages.PlacedTile, 0, len(placements))
	var text strings.Builder

	for _, placement := range placements {
		key := [2]int{placement.Row, placement.Col}
		if seen[key] {
			return board.Frame{}, nil, "", fmt.Errorf("duplicate placement at row %d col %d", placement.Row, placement.Col)
		}
		seen[key] = true

		color := placement.Color
		if color == "" {
			color = defaultColor
		}
		color, err = messages.NormalizeColor(color)
		if err != nil {
			return board.Frame{}, nil, "", err
		}

		cell, err := NormalizeTileCell(placement.Symbol, color)
		if err != nil {
			return board.Frame{}, nil, "", err
		}
		if err := frame.Set(placement.Row, placement.Col, cell); err != nil {
			return board.Frame{}, nil, "", err
		}

		stored = append(stored, messages.PlacedTile{
			Row:    placement.Row,
			Col:    placement.Col,
			Symbol: cell.Symbol,
			Color:  cell.Color,
		})
		text.WriteString(cell.Symbol)
	}

	return frame, stored, text.String(), nil
}

func ExactFrameFromInput(input messages.FrameInput, defaultColor string) (board.Frame, messages.FrameInput, string, error) {
	frame, err := board.NewFrame(board.DefaultRows, board.DefaultCols)
	if err != nil {
		return board.Frame{}, messages.FrameInput{}, "", err
	}

	defaultColor, err = messages.NormalizeColor(defaultColor)
	if err != nil {
		return board.Frame{}, messages.FrameInput{}, "", err
	}

	if len(input.Cells) != board.DefaultRows {
		return board.Frame{}, messages.FrameInput{}, "", fmt.Errorf("frame cells must have %d rows", board.DefaultRows)
	}
	hasColors := len(input.Colors) != 0
	if hasColors && len(input.Colors) != board.DefaultRows {
		return board.Frame{}, messages.FrameInput{}, "", fmt.Errorf("frame colors must have %d rows", board.DefaultRows)
	}

	stored := messages.FrameInput{
		Cells:  make([][]string, board.DefaultRows),
		Colors: make([][]string, board.DefaultRows),
	}
	var textRows []string

	for row := 0; row < board.DefaultRows; row++ {
		if len(input.Cells[row]) != board.DefaultCols {
			return board.Frame{}, messages.FrameInput{}, "", fmt.Errorf("frame cells row %d must have %d columns", row, board.DefaultCols)
		}
		if hasColors && len(input.Colors[row]) != board.DefaultCols {
			return board.Frame{}, messages.FrameInput{}, "", fmt.Errorf("frame colors row %d must have %d columns", row, board.DefaultCols)
		}

		stored.Cells[row] = make([]string, board.DefaultCols)
		stored.Colors[row] = make([]string, board.DefaultCols)
		var textRow strings.Builder

		for col := 0; col < board.DefaultCols; col++ {
			symbol := input.Cells[row][col]
			if symbol == "" {
				symbol = " "
			}

			color := defaultColor
			if hasColors && input.Colors[row][col] != "" {
				color = input.Colors[row][col]
			}
			color, err = messages.NormalizeColor(color)
			if err != nil {
				return board.Frame{}, messages.FrameInput{}, "", err
			}

			cell, err := NormalizeTileCell(symbol, color)
			if err != nil {
				return board.Frame{}, messages.FrameInput{}, "", err
			}
			if err := frame.Set(row, col, cell); err != nil {
				return board.Frame{}, messages.FrameInput{}, "", err
			}

			stored.Cells[row][col] = cell.Symbol
			stored.Colors[row][col] = cell.Color
			textRow.WriteString(cell.Symbol)
		}

		if rowText := strings.TrimSpace(textRow.String()); rowText != "" {
			textRows = append(textRows, rowText)
		}
	}

	if len(textRows) == 0 {
		return board.Frame{}, messages.FrameInput{}, "", fmt.Errorf("frame is empty")
	}

	return frame, stored, strings.Join(textRows, " / "), nil
}

func NormalizeTileCell(symbol, color string) (board.Cell, error) {
	if symbol == " " {
		return board.NewCell(" ", color), nil
	}

	cells, err := board.NormalizeCells(symbol, color)
	if err != nil {
		return board.Cell{}, err
	}
	if len(cells) != 1 {
		return board.Cell{}, fmt.Errorf("tile symbol %q must normalize to exactly one tile", symbol)
	}

	return cells[0], nil
}
