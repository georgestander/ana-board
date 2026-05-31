package board

import (
	"fmt"
	"strings"
)

var allowedColors = []string{"white", "green", "amber", "red", "blue", "violet"}

func AllowedColors() []string {
	return append([]string(nil), allowedColors...)
}

func NormalizeColor(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return DefaultColor, nil
	}

	for _, color := range allowedColors {
		if value == color {
			return value, nil
		}
	}

	return "", fmt.Errorf("color %q is not supported", value)
}

func IsAllowedColor(value string) bool {
	_, err := NormalizeColor(value)
	return err == nil
}
