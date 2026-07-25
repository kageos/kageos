package llms

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ProviderError preserves structured upstream error fields so recovery logic
// does not have to guess solely from a localized error string.
type ProviderError struct {
	HTTPStatus int
	Code       string
	Type       string
	Param      string
	Message    string
}

func (e *ProviderError) Error() string {
	if e == nil {
		return "LLM provider error"
	}
	details := make([]string, 0, 3)
	if strings.TrimSpace(e.Code) != "" {
		details = append(details, "code="+e.Code)
	}
	if strings.TrimSpace(e.Type) != "" {
		details = append(details, "type="+e.Type)
	}
	if strings.TrimSpace(e.Param) != "" {
		details = append(details, "param="+e.Param)
	}
	prefix := "LLM provider error"
	if e.HTTPStatus > 0 {
		prefix = fmt.Sprintf("HTTP %d", e.HTTPStatus)
	}
	if len(details) > 0 {
		prefix += " (" + strings.Join(details, ", ") + ")"
	}
	if strings.TrimSpace(e.Message) != "" {
		return prefix + ": " + strings.TrimSpace(e.Message)
	}
	return prefix
}

// ContextWindowError marks failures caused by an oversized request.
type ContextWindowError struct {
	Message          string
	MaxContextTokens int
	Cause            error
}

func (e *ContextWindowError) Error() string {
	if e == nil || strings.TrimSpace(e.Message) == "" {
		return "LLM context window exceeded"
	}
	return e.Message
}

func (e *ContextWindowError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

var contextWindowNumberPattern = regexp.MustCompile(`(?i)(?:maximum context length|context window|context length)[^0-9]{0,32}([0-9][0-9,]{3,})`)

func IsContextWindowError(err error) bool {
	if err == nil {
		return false
	}
	var contextErr *ContextWindowError
	if errors.As(err, &contextErr) {
		return true
	}
	var providerErr *ProviderError
	if errors.As(err, &providerErr) && IsContextWindowProviderError(providerErr.Code, providerErr.Type, providerErr.Param, providerErr.Message) {
		return true
	}
	return IsContextWindowErrorMessage(err.Error())
}

func IsContextWindowProviderError(code, errorType, param, message string) bool {
	structured := normalizeErrorIdentifier(code) + " " + normalizeErrorIdentifier(errorType)
	for _, strongCode := range []string{
		"context_length_exceeded", "context_window_exceeded", "max_context_length_exceeded",
		"prompt_too_long", "input_too_long", "request_too_large_for_context_window",
	} {
		if strings.Contains(structured, strongCode) {
			return true
		}
	}
	param = strings.ToLower(strings.TrimSpace(param))
	messageLower := strings.ToLower(strings.TrimSpace(message))
	if (param == "messages" || param == "input" || param == "prompt") &&
		(strings.Contains(messageLower, "too many tokens") || strings.Contains(messageLower, "token limit exceeded")) {
		return true
	}
	return IsContextWindowErrorMessage(strings.Join([]string{param, message}, " "))
}

// IsContextWindowErrorMessage intentionally requires context/input wording.
// Generic token errors are excluded because they often mean TPM rate limits,
// account quota, or an invalid output-token setting.
func IsContextWindowErrorMessage(message string) bool {
	text := strings.ToLower(strings.TrimSpace(message))
	if text == "" {
		return false
	}
	for _, strong := range []string{
		"context window exceeds", "context window exceeded", "context length exceeded",
		"maximum context length", "exceeds the context window", "exceed context limit",
		"context size exceeded", "prompt is too long", "prompt too long",
		"input is too long", "input too long", "request is too large for the model",
		"please reduce the length of the messages", "提示词过长",
		"context_length_exceeded", "context_window_exceeded", "prompt_too_long",
	} {
		if strings.Contains(text, strong) {
			return true
		}
	}
	if (strings.Contains(text, "上下文长度") || strings.Contains(text, "输入长度")) &&
		(strings.Contains(text, "超过") || strings.Contains(text, "过长")) {
		return true
	}
	for _, excluded := range []string{
		"rate limit", "tokens per minute", "requests per minute", "tpm", "rpm",
		"quota", "credit", "billing", "insufficient_quota", "payload too large",
		"image", "audio", "file size", "max_tokens parameter", "max output token",
	} {
		if strings.Contains(text, excluded) {
			return false
		}
	}
	return (strings.Contains(text, "input") || strings.Contains(text, "prompt")) &&
		strings.Contains(text, "token") &&
		(strings.Contains(text, "limit") || strings.Contains(text, "maximum") || strings.Contains(text, "too many"))
}

func ContextWindowLimitFromError(err error) int {
	if err == nil {
		return 0
	}
	var contextErr *ContextWindowError
	if errors.As(err, &contextErr) && contextErr.MaxContextTokens > 0 {
		return contextErr.MaxContextTokens
	}
	match := contextWindowNumberPattern.FindStringSubmatch(err.Error())
	if len(match) < 2 {
		return 0
	}
	value, _ := strconv.Atoi(strings.ReplaceAll(match[1], ",", ""))
	return value
}

func normalizeErrorIdentifier(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.NewReplacer("-", "_", " ", "_").Replace(value)
	return value
}

func providerHTTPError(status int, body []byte) error {
	providerErr := &ProviderError{HTTPStatus: status}
	var payload map[string]interface{}
	if json.Unmarshal(body, &payload) == nil {
		root := payload
		if nested, ok := payload["error"].(map[string]interface{}); ok {
			root = nested
		}
		providerErr.Message = stringErrorField(root, "message")
		providerErr.Code = stringErrorField(root, "code")
		providerErr.Type = stringErrorField(root, "type")
		providerErr.Param = stringErrorField(root, "param")
		if providerErr.Type == "" {
			providerErr.Type = stringErrorField(payload, "type")
		}
	}
	if providerErr.Message == "" {
		providerErr.Message = strings.TrimSpace(string(body))
	}
	if IsContextWindowProviderError(providerErr.Code, providerErr.Type, providerErr.Param, providerErr.Message) {
		contextErr := &ContextWindowError{Message: providerErr.Error(), Cause: providerErr}
		contextErr.MaxContextTokens = ContextWindowLimitFromError(contextErr)
		return contextErr
	}
	return providerErr
}

func stringErrorField(values map[string]interface{}, key string) string {
	value, ok := values[key]
	if !ok || value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text)
	}
	return strings.TrimSpace(fmt.Sprint(value))
}
