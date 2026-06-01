package codexbridge

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"strings"
)

type Context struct {
	Project string `json:"project,omitempty"`
	Thread  string `json:"thread,omitempty"`
}

func (context Context) DisplayLabel() string {
	project := strings.TrimSpace(context.Project)
	if project == "" {
		project = "CODEX"
	}
	thread := strings.TrimSpace(context.Thread)
	if thread == "" {
		return project
	}
	label := project + " " + thread
	if len(label) <= 15 {
		return label
	}
	return project
}

func extractContext(fields []textField) Context {
	var context Context
	for _, field := range fields {
		key := strings.ToLower(field.Key)
		text := strings.TrimSpace(field.Text)
		if text == "" {
			continue
		}

		if context.Project == "" && isCWDKey(key) {
			context.Project = projectLabelFromPath(text)
			continue
		}
		if context.Thread == "" && isThreadKey(key) {
			context.Thread = "T" + shortHash(text, 4)
		}
	}
	return context
}

func isCWDKey(key string) bool {
	for _, part := range strings.Split(key, ".") {
		switch part {
		case "cwd", "workdir", "working_directory", "current_directory", "current_dir":
			return true
		}
	}
	return false
}

func isThreadKey(key string) bool {
	if strings.Contains(key, "prompt") || strings.Contains(key, "message") || strings.Contains(key, "last_message") {
		return false
	}
	return strings.Contains(key, "thread") || strings.Contains(key, "session") || strings.Contains(key, "conversation")
}

func projectLabelFromPath(path string) string {
	cleaned := strings.TrimRight(strings.TrimSpace(path), "/")
	if cleaned == "" {
		return ""
	}
	base := filepath.Base(cleaned)
	if base == "." || base == "/" || base == "" {
		return ""
	}
	return compactLabel(base, 8)
}

func compactLabel(value string, max int) string {
	tokens := labelTokens(value)
	if len(tokens) == 0 {
		return ""
	}
	if len(tokens) > 1 && isNumeric(tokens[1]) && len(tokens[0])+len(tokens[1]) <= max {
		return tokens[0] + tokens[1]
	}
	if len(tokens[0]) >= 3 {
		return trimLabel(tokens[0], max)
	}

	var joined strings.Builder
	for _, token := range tokens {
		if joined.Len()+len(token) > max {
			break
		}
		joined.WriteString(token)
	}
	if joined.Len() == 0 {
		return trimLabel(tokens[0], max)
	}
	return joined.String()
}

func labelTokens(value string) []string {
	value = strings.ToUpper(value)
	var tokens []string
	var current strings.Builder
	for _, char := range value {
		if (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') {
			current.WriteRune(char)
			continue
		}
		if current.Len() > 0 {
			tokens = append(tokens, current.String())
			current.Reset()
		}
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}
	return tokens
}

func trimLabel(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}

func isNumeric(value string) bool {
	if value == "" {
		return false
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

func shortHash(value string, length int) string {
	sum := sha256.Sum256([]byte(value))
	encoded := strings.ToUpper(hex.EncodeToString(sum[:]))
	if len(encoded) < length {
		return encoded
	}
	return encoded[:length]
}
