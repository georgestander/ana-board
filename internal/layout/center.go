package layout

import (
	"fmt"
	"strings"

	"github.com/georgestander/ana-board/internal/board"
)

func CenterText(text string) (board.Frame, error) {
	return CenterTextWithColor(text, board.DefaultColor)
}

func CenterTextWithColor(text, color string) (board.Frame, error) {
	cells, err := board.NormalizeCells(text, color)
	if err != nil {
		return board.Frame{}, err
	}

	return CenterCells(cells)
}

func CenterSegments(segments []board.TextSegment, defaultColor string) (board.Frame, error) {
	cells, err := board.NormalizeSegmentCells(segments, defaultColor)
	if err != nil {
		return board.Frame{}, err
	}

	return CenterCells(cells)
}

func CenterCells(cells []board.Cell) (board.Frame, error) {
	lines, err := wrapCellLines(cells)
	if err != nil {
		return board.Frame{}, err
	}

	if len(lines) > board.DefaultRows {
		return board.Frame{}, fmt.Errorf("message needs %d rows but board only has %d", len(lines), board.DefaultRows)
	}

	frame, err := board.NewFrame(board.DefaultRows, board.DefaultCols)
	if err != nil {
		return board.Frame{}, err
	}

	startRow := (board.DefaultRows - len(lines)) / 2

	for lineIndex, line := range lines {
		row := startRow + lineIndex
		col := (board.DefaultCols - len(line)) / 2

		for offset, cell := range line {
			if err := frame.Set(row, col+offset, cell); err != nil {
				return board.Frame{}, err
			}
		}
	}

	return frame, nil
}

func wrapWords(text string) ([]string, error) {
	symbols, err := board.NormalizeSymbols(text)
	if err != nil {
		return nil, err
	}

	cells := make([]board.Cell, len(symbols))
	for index, symbol := range symbols {
		cells[index] = board.NewCell(symbol, board.DefaultColor)
	}

	symbolLines, err := wrapCellLines(cells)
	if err != nil {
		return nil, err
	}

	lines := make([]string, len(symbolLines))
	for index, line := range symbolLines {
		lines[index] = cellsToString(line)
	}

	return lines, nil
}

func wrapCellLines(cells []board.Cell) ([][]board.Cell, error) {
	words := splitWords(cells)
	var lines [][]board.Cell
	var currentLine []board.Cell

	for _, word := range words {
		if len(word) > board.DefaultCols {
			return nil, fmt.Errorf("word %q is too long for one board row", cellsToString(word))
		}

		if len(currentLine) == 0 {
			currentLine = append([]board.Cell(nil), word...)
			continue
		}

		candidateLen := len(currentLine) + 1 + len(word)
		if candidateLen <= board.DefaultCols {
			currentLine = append(currentLine, board.NewCell(" ", board.DefaultColor))
			currentLine = append(currentLine, word...)
			continue
		}

		lines = append(lines, currentLine)
		currentLine = append([]board.Cell(nil), word...)
	}

	if len(currentLine) != 0 {
		lines = append(lines, currentLine)
	}

	return lines, nil
}

func splitWords(cells []board.Cell) [][]board.Cell {
	var words [][]board.Cell
	var current []board.Cell

	for _, cell := range cells {
		if cell.Symbol == " " {
			if len(current) != 0 {
				words = append(words, current)
				current = nil
			}
			continue
		}

		current = append(current, cell)
	}

	if len(current) != 0 {
		words = append(words, current)
	}

	return words
}

func cellsToString(cells []board.Cell) string {
	var builder strings.Builder
	for _, cell := range cells {
		builder.WriteString(cell.Symbol)
	}

	return builder.String()
}
