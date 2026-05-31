package messages

import "time"

type Message struct {
	ID         string       `json:"id"`
	Text       string       `json:"text"`
	Segments   []Segment    `json:"segments,omitempty"`
	Tiles      []Tile       `json:"tiles,omitempty"`
	Placements []PlacedTile `json:"placements,omitempty"`
	Frame      *FrameInput  `json:"frame,omitempty"`
	Source     string       `json:"source"`
	Priority   string       `json:"priority"`
	Animation  string       `json:"animation"`
	Kind       string       `json:"kind"`
	Color      string       `json:"color"`
	CreatedAt  time.Time    `json:"created_at"`
	Status     string       `json:"status"`
}

type SubmitRequest struct {
	Text       string       `json:"text"`
	Segments   []Segment    `json:"segments,omitempty"`
	Tiles      []Tile       `json:"tiles,omitempty"`
	Placements []PlacedTile `json:"placements,omitempty"`
	Frame      *FrameInput  `json:"frame,omitempty"`
	Source     string       `json:"source"`
	Priority   string       `json:"priority"`
	Animation  string       `json:"animation"`
	Kind       string       `json:"kind"`
	Color      string       `json:"color"`
}

type Segment struct {
	Text  string `json:"text"`
	Color string `json:"color,omitempty"`
}

type Tile struct {
	Symbol string `json:"symbol"`
	Color  string `json:"color,omitempty"`
}

type PlacedTile struct {
	Row    int    `json:"row"`
	Col    int    `json:"col"`
	Symbol string `json:"symbol"`
	Color  string `json:"color,omitempty"`
}

type FrameInput struct {
	Cells  [][]string `json:"cells"`
	Colors [][]string `json:"colors,omitempty"`
}
