package board

import "testing"

func TestIsAllowedChar(t *testing.T) {
	tests := []struct {
		name string
		char rune
		want bool
	}{
		{
			name: "uppercase letter",
			char: 'A',
			want: true,
		},
		{
			name: "digit",
			char: '7',
			want: true,
		},
		{
			name: "space",
			char: ' ',
			want: true,
		},
		{
			name: "punctuation",
			char: '!',
			want: true,
		},
		{
			name: "lowercase letter",
			char: 'a',
			want: false,
		},
		{
			name: "unsupported symbol",
			char: '@',
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsAllowedChar(tt.char)

			if got != tt.want {
				t.Fatalf("IsAllowedChar(%q) = %v, want %v", tt.char, got, tt.want)
			}
		})
	}
}

func TestNormalizeTextUppercasesAndReplacesUnsupportedChars(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "uppercase lowercase letters",
			input: "hello",
			want:  "HELLO",
		},
		{
			name:  "keeps allowed punctuation",
			input: "hi, ana!",
			want:  "HI, ANA!",
		},
		{
			name:  "replaces unsupported symbols",
			input: "price is R100 @ noon",
			want:  "PRICE IS R100 ? NOON",
		},
		{
			name:  "keeps native emoji",
			input: "done ✅",
			want:  "DONE ✅",
		},
		{
			name:  "expands named emoji alias",
			input: "hello :globe:",
			want:  "HELLO 🌍",
		},
		{
			name:  "collapses repeated spaces",
			input: "hello   ana",
			want:  "HELLO ANA",
		},
		{
			name:  "trims and normalizes whitespace",
			input: "\n hello\t\tana   board \n",
			want:  "HELLO ANA BOARD",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeText(tt.input)
			if err != nil {
				t.Fatalf("NormalizeText returned error: %v", err)
			}

			if got != tt.want {
				t.Fatalf("NormalizeText(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeSymbolsTreatsEmojiAsOneTile(t *testing.T) {
	got, err := NormalizeSymbols("hi :globe:")
	if err != nil {
		t.Fatalf("NormalizeSymbols returned error: %v", err)
	}

	if len(got) != 4 {
		t.Fatalf("len(got) = %d, want 4", len(got))
	}

	if got[3] != "🌍" {
		t.Fatalf("last symbol = %q, want globe emoji", got[3])
	}
}

func TestNormalizeSymbolsTreatsNativeEmojiGraphemesAsOneTile(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "skin tone", input: "ok 👍🏽", want: "👍🏽"},
		{name: "family zwj sequence", input: "family 👨‍👩‍👧‍👦", want: "👨‍👩‍👧‍👦"},
		{name: "flag pair", input: "za 🇿🇦", want: "🇿🇦"},
		{name: "keycap", input: "one 1️⃣", want: "1️⃣"},
		{name: "heart variation", input: "love ❤️", want: "❤️"},
		{name: "profession zwj sequence", input: "ship 👩🏽‍💻", want: "👩🏽‍💻"},
		{name: "arrow variation", input: "next ↗️", want: "↗️"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeSymbols(tt.input)
			if err != nil {
				t.Fatalf("NormalizeSymbols returned error: %v", err)
			}

			if got[len(got)-1] != tt.want {
				t.Fatalf("last symbol = %q, want %q; all=%q", got[len(got)-1], tt.want, got)
			}
		})
	}
}

func TestNormalizeColoredSymbolsSupportsInlineColorTokens(t *testing.T) {
	got, err := NormalizeSegmentCells([]TextSegment{{Text: "ok [green]go [red]stop [/]done", Color: "white"}}, "white")
	if err != nil {
		t.Fatalf("NormalizeSegmentCells returned error: %v", err)
	}

	if got[0].Symbol != "O" || got[0].Color != "white" {
		t.Fatalf("first cell = %#v, want white O", got[0])
	}

	if got[3].Symbol != "G" || got[3].Color != "green" {
		t.Fatalf("green cell = %#v, want green G", got[3])
	}

	if got[6].Symbol != "S" || got[6].Color != "red" {
		t.Fatalf("red cell = %#v, want red S", got[6])
	}

	if got[11].Symbol != "D" || got[11].Color != "white" {
		t.Fatalf("reset cell = %#v, want white D", got[11])
	}
}

func TestNormalizeSymbolsKeepsUnknownEmojiAliasAsText(t *testing.T) {
	got, err := NormalizeText("hello :unknown:")
	if err != nil {
		t.Fatalf("NormalizeText returned error: %v", err)
	}

	if got != "HELLO :UNKNOWN:" {
		t.Fatalf("NormalizeText returned %q, want %q", got, "HELLO :UNKNOWN:")
	}
}

func TestNormalizeColoredSymbolsRejectsUnknownInlineColor(t *testing.T) {
	_, err := NormalizeCells("hello [neon]world", "white")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNormalizeTextRejectsEmptyMessages(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "empty string",
			input: "",
		},
		{
			name:  "spaces only",
			input: "   ",
		},
		{
			name:  "tabs and newlines only",
			input: "\n\t\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NormalizeText(tt.input)

			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
