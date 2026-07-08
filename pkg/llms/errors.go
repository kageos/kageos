package llms

import (
	"errors"
	"strings"
)

// ContextWindowError marks provider failures caused by an oversized request.
type ContextWindowError struct {
	Message string
}

func (e *ContextWindowError) Error() string {
	if strings.TrimSpace(e.Message) == "" {
		return "LLM context window exceeded"
	}
	return e.Message
}

// IsContextWindowError reports whether err is a provider context-window failure.
func IsContextWindowError(err error) bool {
	if err == nil {
		return false
	}
	var contextErr *ContextWindowError
	if errors.As(err, &contextErr) {
		return true
	}
	return IsContextWindowErrorMessage(err.Error())
}

// IsContextWindowErrorMessage matches common OpenAI/Anthropic/proxy wording.
func IsContextWindowErrorMessage(message string) bool {
	text := strings.ToLower(strings.TrimSpace(message))
	if text == "" {
		return false
	}
	patterns := []string{
		"context window exceeds",
		"context window exceeded",
		"context length exceeded",
		"maximum context length",
		"exceeds the context window",
		"exceed context limit",
		"tokens exceed",
		"too many tokens",
		"input is too long",
	}
	for _, pattern := range patterns {
		if strings.Contains(text, pattern) {
			return true
		}
	}
	return false
}
