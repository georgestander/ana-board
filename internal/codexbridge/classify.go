package codexbridge

import (
	"encoding/json"
	"fmt"
	"strings"
)

type textField struct {
	Key  string
	Text string
}

func extractTextFields(payload []byte) ([]textField, error) {
	payload = []byte(strings.TrimSpace(string(payload)))
	if len(payload) == 0 {
		return nil, nil
	}

	var decoded any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return []textField{{Key: "raw", Text: string(payload)}}, nil
	}

	var fields []textField
	collectTextFields(decoded, "", &fields)
	return fields, nil
}

func collectTextFields(value any, key string, fields *[]textField) {
	switch typed := value.(type) {
	case map[string]any:
		for childKey, childValue := range typed {
			path := childKey
			if key != "" {
				path = key + "." + childKey
			}
			collectTextFields(childValue, path, fields)
		}
	case []any:
		for _, childValue := range typed {
			collectTextFields(childValue, key, fields)
		}
	case string:
		text := strings.TrimSpace(typed)
		if text != "" {
			*fields = append(*fields, textField{Key: strings.ToLower(key), Text: text})
		}
	case fmt.Stringer:
		text := strings.TrimSpace(typed.String())
		if text != "" {
			*fields = append(*fields, textField{Key: strings.ToLower(key), Text: text})
		}
	}
}

func classify(eventName string, fields []textField) Signal {
	lowerEvent := strings.ToLower(eventName)
	signal := Signal{}

	if strings.Contains(lowerEvent, "permission") {
		signal.Approval = true
	}

	for _, field := range fields {
		lowerText := strings.ToLower(field.Text)
		if signal.Swear == "" && isPromptLikeEvent(lowerEvent, field.Key) {
			signal.Swear = detectedSwear(lowerText)
		}
		if !signal.Celebration && isPromptLikeEvent(lowerEvent, field.Key) && containsAny(lowerText, celebrationNeedles) {
			signal.Celebration = true
		}
		if signal.Failure == "" && containsFailure(lowerText) {
			signal.Failure = failureKind(lowerText)
		}
		if signal.Success == "" && containsSuccess(lowerText) {
			signal.Success = successKind(lowerText)
		}
		if !signal.Approval && containsAny(lowerText, approvalNeedles) {
			signal.Approval = true
		}
	}

	return signal
}

func isPromptLikeEvent(eventName, key string) bool {
	if strings.Contains(eventName, "prompt") || strings.Contains(eventName, "user") {
		return true
	}

	for _, part := range []string{"prompt", "user", "input", "message", "text"} {
		if strings.Contains(key, part) {
			return true
		}
	}
	return false
}

func detectedSwear(text string) string {
	for _, word := range []string{"fucking", "fuck", "wtf", "bullshit", "shit", "damn"} {
		if containsTokenish(text, word) {
			if word == "fucking" {
				return "fuck"
			}
			return word
		}
	}
	return ""
}

func containsFailure(text string) bool {
	if containsAny(text, []string{"no error", "no errors", "without error", "without errors"}) {
		return false
	}
	if containsAny(text, []string{"test fail", "tests fail", "build fail", "failed", "failure", "panic", "exception", "timed out", "timeout"}) {
		return true
	}
	if strings.Contains(text, " error") || strings.HasPrefix(text, "error") {
		return true
	}
	return false
}

func failureKind(text string) string {
	if strings.Contains(text, "test") {
		return "test"
	}
	if strings.Contains(text, "build") {
		return "build"
	}
	return "general"
}

func containsSuccess(text string) bool {
	if containsAny(text, []string{"not done", "not complete", "not completed"}) {
		return false
	}
	return containsAny(text, []string{
		"test passed",
		"tests passed",
		"build passed",
		"passed",
		"implemented",
		"complete",
		"completed",
		"fixed",
		"done",
		"deployed",
		"pushed",
		"commit",
	})
}

func successKind(text string) string {
	if strings.Contains(text, "test") {
		return "test"
	}
	if strings.Contains(text, "build") {
		return "build"
	}
	return "general"
}

func containsAny(text string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}

func containsTokenish(text, needle string) bool {
	index := strings.Index(text, needle)
	for index >= 0 {
		beforeOK := index == 0 || !isAlphaNum(text[index-1])
		afterIndex := index + len(needle)
		afterOK := afterIndex >= len(text) || !isAlphaNum(text[afterIndex])
		if beforeOK && afterOK {
			return true
		}
		next := strings.Index(text[index+len(needle):], needle)
		if next < 0 {
			return false
		}
		index += len(needle) + next
	}
	return false
}

func isAlphaNum(char byte) bool {
	return (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9')
}

var celebrationNeedles = []string{
	"fuck yes",
	"f yeah",
	"hell yes",
	"nice",
	"great",
	"awesome",
	"love it",
	"ship it",
}

var approvalNeedles = []string{
	"approval required",
	"approval needed",
	"permission required",
	"permission needed",
	"needs approval",
	"confirm",
}
