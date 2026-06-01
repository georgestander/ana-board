package board

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/rivo/uniseg"
)

var allowedChars = map[rune]bool{
	'A':  true,
	'B':  true,
	'C':  true,
	'D':  true,
	'E':  true,
	'F':  true,
	'G':  true,
	'H':  true,
	'I':  true,
	'J':  true,
	'K':  true,
	'L':  true,
	'M':  true,
	'N':  true,
	'O':  true,
	'P':  true,
	'Q':  true,
	'R':  true,
	'S':  true,
	'T':  true,
	'U':  true,
	'V':  true,
	'W':  true,
	'X':  true,
	'Y':  true,
	'Z':  true,
	'0':  true,
	'1':  true,
	'2':  true,
	'3':  true,
	'4':  true,
	'5':  true,
	'6':  true,
	'7':  true,
	'8':  true,
	'9':  true,
	' ':  true,
	'.':  true,
	',':  true,
	'!':  true,
	'?':  true,
	':':  true,
	'-':  true,
	'/':  true,
	'\'': true,
	'"':  true,
	'█':  true,
}

var emojiAliases = map[string]string{
	"alert":    "🚨",
	"approval": "✋",
	"build":    "🛠️",
	"calendar": "📅",
	"check":    "✅",
	"clock":    "⏰",
	"code":     "💻",
	"deploy":   "🚢",
	"email":    "✉️",
	"error":    "❌",
	"globe":    "🌍",
	"hand":     "✋",
	"home":     "🏠",
	"idea":     "💡",
	"info":     "ℹ️",
	"lock":     "🔒",
	"mail":     "✉️",
	"money":    "💰",
	"pin":      "📌",
	"reminder": "⏰",
	"rocket":   "🚀",
	"ship":     "🚢",
	"success":  "✅",
	"task":     "📌",
	"tools":    "🛠️",
	"user":     "👤",
	"warning":  "⚠️",
	"world":    "🌍",
}

type TextSegment struct {
	Text  string
	Color string
}

type NormalizedSymbol struct {
	Symbol string
	Color  string
}

func IsAllowedChar(char rune) bool {
	return allowedChars[char]
}

func AllowedEmojiNames() []string {
	names := make([]string, 0, len(emojiAliases))
	for name := range emojiAliases {
		names = append(names, name)
	}

	sort.Strings(names)
	return names
}

func EmojiAliases() map[string]string {
	emojis := make(map[string]string, len(emojiAliases))
	for name, emoji := range emojiAliases {
		emojis[name] = emoji
	}

	return emojis
}

func AllowedEmojis() map[string]string {
	return EmojiAliases()
}

func EmojiByName(name string) (string, bool) {
	emoji, ok := emojiAliases[strings.ToLower(strings.TrimSpace(name))]
	return emoji, ok
}

func NormalizeText(input string) (string, error) {
	symbols, err := NormalizeSymbols(input)
	if err != nil {
		return "", err
	}

	return strings.Join(symbols, ""), nil
}

func NormalizeSymbols(input string) ([]string, error) {
	colored, err := NormalizeColoredSymbols([]TextSegment{{Text: input}})
	if err != nil {
		return nil, err
	}

	symbols := make([]string, len(colored))
	for index, symbol := range colored {
		symbols[index] = symbol.Symbol
	}

	return symbols, nil
}

func NormalizeCells(input, color string) ([]Cell, error) {
	return NormalizeSegmentCells([]TextSegment{{Text: input, Color: color}}, color)
}

func NormalizeSegmentCells(segments []TextSegment, defaultColor string) ([]Cell, error) {
	if len(segments) == 0 {
		return nil, fmt.Errorf("text is required")
	}

	for index := range segments {
		if segments[index].Color == "" {
			segments[index].Color = defaultColor
		}
		color, err := NormalizeColor(segments[index].Color)
		if err != nil {
			return nil, err
		}
		segments[index].Color = color
	}

	symbols, err := NormalizeColoredSymbols(segments)
	if err != nil {
		return nil, err
	}

	cells := make([]Cell, len(symbols))
	for index, symbol := range symbols {
		cells[index] = NewCell(symbol.Symbol, symbol.Color)
	}

	return cells, nil
}

func NormalizeColoredSymbols(segments []TextSegment) ([]NormalizedSymbol, error) {
	var normalized []NormalizedSymbol
	previousWasSpace := true

	for _, segment := range segments {
		color := segment.Color
		remaining := segment.Text

		for len(remaining) > 0 {
			if nextColor, consumed, ok, err := readInlineColorToken(remaining); ok || err != nil {
				if err != nil {
					return nil, err
				}
				color = nextColor
				remaining = remaining[consumed:]
				continue
			}

			if emoji, consumed, ok := readKnownEmojiAlias(remaining); ok {
				normalized = append(normalized, NormalizedSymbol{Symbol: emoji, Color: color})
				previousWasSpace = false
				remaining = remaining[consumed:]
				continue
			}

			cluster, consumed := nextGrapheme(remaining)
			if cluster == "" {
				break
			}
			remaining = remaining[consumed:]

			if isIgnorableCluster(cluster) {
				continue
			}

			if isSpaceCluster(cluster) {
				if !previousWasSpace {
					normalized = append(normalized, NormalizedSymbol{Symbol: " ", Color: color})
					previousWasSpace = true
				}

				continue
			}

			if isEmojiGrapheme(cluster) {
				normalized = append(normalized, NormalizedSymbol{Symbol: cluster, Color: color})
				previousWasSpace = false
				continue
			}

			normalized = append(normalized, NormalizedSymbol{Symbol: normalizePlainCluster(cluster), Color: color})
			previousWasSpace = false
		}
	}

	if len(normalized) > 0 && normalized[len(normalized)-1].Symbol == " " {
		normalized = normalized[:len(normalized)-1]
	}

	if len(normalized) == 0 {
		return nil, fmt.Errorf("message is empty after normalization")
	}

	return normalized, nil
}

func isEmojiName(name string) bool {
	for _, char := range name {
		if unicode.IsLetter(char) || unicode.IsDigit(char) || char == '-' || char == '_' {
			continue
		}

		return false
	}

	return true
}

func readInlineColorToken(input string) (string, int, bool, error) {
	if !strings.HasPrefix(input, "[") {
		return "", 0, false, nil
	}

	end := strings.Index(input, "]")
	if end <= 0 {
		return "", 0, false, nil
	}

	name := input[1:end]
	if !isColorTokenName(name) {
		return "", 0, false, nil
	}

	if name == "/" || strings.EqualFold(name, "reset") {
		return DefaultColor, end + 1, true, nil
	}

	color, err := NormalizeColor(name)
	if err != nil {
		return "", 0, true, err
	}

	return color, end + 1, true, nil
}

func isColorTokenName(name string) bool {
	if name == "/" {
		return true
	}

	return isEmojiName(name)
}

func readKnownEmojiAlias(input string) (string, int, bool) {
	if !strings.HasPrefix(input, ":") {
		return "", 0, false
	}

	end := strings.Index(input[1:], ":")
	if end < 0 {
		return "", 0, false
	}

	end++
	name := input[1:end]
	if !isEmojiName(name) {
		return "", 0, false
	}

	emoji, ok := EmojiByName(name)
	if !ok {
		return "", 0, false
	}

	return emoji, end + 1, true
}

func nextGrapheme(input string) (string, int) {
	graphemes := uniseg.NewGraphemes(input)
	if !graphemes.Next() {
		return "", 0
	}

	cluster := graphemes.Str()
	return cluster, len(cluster)
}

func isIgnorableCluster(cluster string) bool {
	if cluster == "" {
		return true
	}

	for _, char := range cluster {
		if !isVariationSelector(char) {
			return false
		}
	}

	return true
}

func isSpaceCluster(cluster string) bool {
	for _, char := range cluster {
		if !unicode.IsSpace(char) {
			return false
		}
	}

	return true
}

func isEmojiGrapheme(cluster string) bool {
	if cluster == "" {
		return false
	}

	if strings.ContainsRune(cluster, '\u200d') {
		return true
	}

	for _, char := range cluster {
		if unicode.Is(unicode.So, char) || isEmojiStart(char) || isEmojiModifier(char) || isCombiningKeycap(char) || isRegionalIndicator(char) || isTagRune(char) {
			return true
		}
		if isVariationSelector(char) && len([]rune(cluster)) > 1 {
			return true
		}
	}

	return false
}

func normalizePlainCluster(cluster string) string {
	runes := []rune(cluster)
	if len(runes) != 1 {
		return "?"
	}

	char := unicode.ToUpper(runes[0])
	if IsAllowedChar(char) {
		return string(char)
	}

	return "?"
}

func isEmojiStart(char rune) bool {
	switch {
	case char >= 0x1F000 && char <= 0x1FAFF:
		return true
	case char >= 0x2600 && char <= 0x27BF:
		return true
	case char >= 0x2300 && char <= 0x23FF:
		return true
	case char >= 0x2B00 && char <= 0x2BFF:
		return true
	case char == 0x00A9 || char == 0x00AE || char == 0x203C || char == 0x2049 || char == 0x2122 || char == 0x2139 || char == 0x3030 || char == 0x303D || char == 0x3297 || char == 0x3299:
		return true
	default:
		return false
	}
}

func startsKeycapSequence(runes []rune, start int) bool {
	if start >= len(runes) {
		return false
	}
	char := runes[start]
	if !(char >= '0' && char <= '9') && char != '#' && char != '*' {
		return false
	}
	if start+1 < len(runes) && isCombiningKeycap(runes[start+1]) {
		return true
	}
	return start+2 < len(runes) && isVariationSelector(runes[start+1]) && isCombiningKeycap(runes[start+2])
}

func isVariationSelector(char rune) bool {
	return char == '\ufe0e' || char == '\ufe0f'
}

func isEmojiModifier(char rune) bool {
	return char >= 0x1F3FB && char <= 0x1F3FF
}

func isCombiningKeycap(char rune) bool {
	return char == 0x20E3
}

func isRegionalIndicator(char rune) bool {
	return char >= 0x1F1E6 && char <= 0x1F1FF
}

func isTagRune(char rune) bool {
	return char >= 0xE0020 && char <= 0xE007F
}
