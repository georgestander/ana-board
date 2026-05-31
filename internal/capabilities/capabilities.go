package capabilities

import (
	"github.com/georgestander/ana-board/internal/board"
	"github.com/georgestander/ana-board/internal/messages"
)

type Capabilities struct {
	Board        BoardInfo                `json:"board"`
	Text         TextInfo                 `json:"text"`
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

func Current() Capabilities {
	return Capabilities{
		Board: BoardInfo{
			Rows:     board.DefaultRows,
			Cols:     board.DefaultCols,
			MaxTiles: board.DefaultRows * board.DefaultCols,
		},
		Text: TextInfo{
			AllowedCharacters: `A-Z 0-9 space . , ! ? : - / ' " plus native emoji grapheme clusters`,
			EmojiSupport:      "Native iOS/macOS emoji are accepted directly. There is no emoji whitelist; each visible grapheme cluster counts as one tile. Aliases such as :rocket: are optional shortcuts only.",
			ColorSyntax:       "Each tile can have its own color. Use tiles JSON for exact per-letter color, or inline tokens such as [green]A[amber]N[red]A for quick text. The message color field is only the default for untagged tiles.",
			BestPractice:      "Use row animation. Keep updates short, concrete, and useful. Use per-tile color and native emoji as signal, not decoration.",
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
		},
	}
}
