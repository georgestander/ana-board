package capabilities

import (
	"github.com/georgestander/ana-board/internal/art"
	"github.com/georgestander/ana-board/internal/board"
	"github.com/georgestander/ana-board/internal/messages"
)

type Capabilities struct {
	Board        BoardInfo                `json:"board"`
	Text         TextInfo                 `json:"text"`
	ExactFrame   ExactFrameInfo           `json:"exact_frame"`
	BlockArt     BlockArtInfo             `json:"block_art"`
	Colors       []string                 `json:"colors"`
	Kinds        []string                 `json:"kinds"`
	Animations   []string                 `json:"animations"`
	Priorities   []string                 `json:"priorities"`
	NativeEmoji  bool                     `json:"native_emoji"`
	EmojiAliases map[string]string        `json:"emoji_aliases"`
	Examples     []messages.SubmitRequest `json:"examples"`
}

type BoardInfo struct {
	Rows     int `json:"rows"`
	Cols     int `json:"cols"`
	MaxTiles int `json:"max_tiles"`
}

type TextInfo struct {
	AllowedCharacters string `json:"allowed_characters"`
	EmojiSupport      string `json:"emoji_support"`
	ColorSyntax       string `json:"color_syntax"`
	BestPractice      string `json:"best_practice"`
}

type ExactFrameInfo struct {
	PlacementSyntax string `json:"placement_syntax"`
	FrameSyntax     string `json:"frame_syntax"`
	TimeSyntax      string `json:"time_syntax"`
}

type BlockArtInfo struct {
	PixelSymbol string   `json:"pixel_symbol"`
	Sprites     []string `json:"sprites"`
	CLISyntax   string   `json:"cli_syntax"`
	MCPSyntax   string   `json:"mcp_syntax"`
}

func Current() Capabilities {
	return Capabilities{
		Board: BoardInfo{
			Rows:     board.DefaultRows,
			Cols:     board.DefaultCols,
			MaxTiles: board.DefaultRows * board.DefaultCols,
		},
		Text: TextInfo{
			AllowedCharacters: `A-Z 0-9 space . , ! ? : - / ' " █ plus native emoji grapheme clusters`,
			EmojiSupport:      "Native iOS/macOS emoji are accepted directly. There is no emoji whitelist; each visible grapheme cluster counts as one tile. Aliases such as :rocket: are optional shortcuts only.",
			ColorSyntax:       "Each tile can have its own color. Use tiles JSON for exact per-letter color, inline tokens such as [green]A[amber]N[red]A for quick text, or block-art frames made from colored █ pixels. The message color field is only the default for untagged tiles.",
			BestPractice:      "Use row animation. Keep updates short, concrete, and useful. Use sprites for tiny block art, and placements or frame when exact row and column control matters.",
		},
		ExactFrame: ExactFrameInfo{
			PlacementSyntax: `Use placements JSON for sparse exact control: [{"row":0,"col":0,"symbol":"A","color":"green"}]. Rows are 0-9 and columns are 0-21.`,
			FrameSyntax:     "Use frame JSON for a full exact board: cells is 10 rows x 22 columns; colors is optional and must be the same shape when provided.",
			TimeSyntax:      `ana-boardctl supports optional client-side exact-time sending with --at "2026-05-31T18:30:00+02:00" or the scripts/ana-board-at helper.`,
		},
		BlockArt: BlockArtInfo{
			PixelSymbol: art.PixelSymbol,
			Sprites:     art.ListSprites(),
			CLISyntax:   `ana-boardctl send --sprite trophy or ana-boardctl frame --image ./tiny.png`,
			MCPSyntax:   `Use ana_board_list_sprites, ana_board_preview_sprite, and ana_board_send_sprite for named block-art sprites.`,
		},
		Colors:       messages.AllowedColors(),
		Kinds:        messages.AllowedKinds(),
		Animations:   messages.AllowedAnimations(),
		Priorities:   messages.AllowedPriorities(),
		NativeEmoji:  true,
		EmojiAliases: board.EmojiAliases(),
		Examples: []messages.SubmitRequest{
			{Text: "[green]B[amber]U[blue]I[violet]L[green]D PASSED ✅", Source: "codex", Priority: messages.DefaultPriority, Kind: "success", Color: "white", Animation: "row"},
			{Text: "[amber]EMAIL NEEDS REPLY ✉️", Source: "hermes", Priority: messages.DefaultPriority, Kind: "email", Color: "white", Animation: "row"},
			{Text: "[blue]D[green]E[amber]P[red]L[violet]O[blue]Y COMPLETE 🚀", Source: "ci", Priority: messages.DefaultPriority, Kind: "deploy", Color: "white", Animation: "row"},
			{Placements: []messages.PlacedTile{{Row: 0, Col: 0, Symbol: "A", Color: "green"}, {Row: board.DefaultRows - 1, Col: board.DefaultCols - 1, Symbol: "✅", Color: "blue"}}, Source: "codex", Priority: messages.DefaultPriority, Kind: "info", Color: "white", Animation: "row"},
		},
	}
}
