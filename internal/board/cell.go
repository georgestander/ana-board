package board

type Cell struct {
	Symbol string
	Color  string
}

func NewCell(symbol, color string) Cell {
	if symbol == "" {
		symbol = " "
	}
	if color == "" {
		color = DefaultColor
	}

	return Cell{
		Symbol: symbol,
		Color:  color,
	}
}
