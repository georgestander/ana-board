package messages

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/georgestander/ana-board/internal/board"
)

const (
	DefaultAnimation = "row"
	DefaultColor     = "white"
	DefaultKind      = "info"
	DefaultPriority  = "normal"
	DefaultSource    = "unknown"
)

var allowedAnimations = []string{"row"}
var allowedKinds = []string{"info", "success", "warning", "error", "reminder", "email", "task", "deploy", "build"}
var allowedPriorities = []string{"low", "normal", "high"}

func AllowedAnimations() []string {
	return append([]string(nil), allowedAnimations...)
}

func AllowedColors() []string {
	return board.AllowedColors()
}

func AllowedKinds() []string {
	return append([]string(nil), allowedKinds...)
}

func AllowedPriorities() []string {
	return append([]string(nil), allowedPriorities...)
}

func NormalizeAnimation(value string) (string, error) {
	return normalizeAllowed(value, DefaultAnimation, allowedAnimations, "animation")
}

func NormalizeColor(value string) (string, error) {
	return board.NormalizeColor(value)
}

func NormalizeKind(value string) (string, error) {
	return normalizeAllowed(value, DefaultKind, allowedKinds, "kind")
}

func NormalizePriority(value string) (string, error) {
	return normalizeAllowed(value, DefaultPriority, allowedPriorities, "priority")
}

func NormalizeSource(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return DefaultSource, nil
	}
	if len(value) > 64 {
		return "", fmt.Errorf("source must be 64 characters or fewer")
	}

	for _, char := range value {
		if unicode.IsLetter(char) || unicode.IsDigit(char) || char == '-' || char == '_' || char == '.' {
			continue
		}

		return "", fmt.Errorf("source contains unsupported character %q", char)
	}

	return value, nil
}

func normalizeAllowed(value, fallback string, allowed []string, label string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return fallback, nil
	}

	for _, candidate := range allowed {
		if value == candidate {
			return value, nil
		}
	}

	return "", fmt.Errorf("%s %q is not supported", label, value)
}
