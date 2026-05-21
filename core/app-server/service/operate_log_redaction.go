package service

import (
	"encoding/json"
	"regexp"
	"strings"
)

var sensitiveOperateLogKeyPattern = regexp.MustCompile(`(?i)(password|passwd|pwd|token|secret|api[_-]?key|access[_-]?key|authorization|auth[_-]?token|cookie|session|credential|private[_-]?key)`)

type operateLogRedactor struct {
	sensitiveFields map[string]struct{}
}

type operateLogRedactionResult struct {
	value   interface{}
	removed bool
}

func newOperateLogRedactor(sensitiveFields map[string]struct{}) operateLogRedactor {
	return operateLogRedactor{sensitiveFields: sensitiveFields}
}

func sanitizeOperateLogRawMessage(raw json.RawMessage) json.RawMessage {
	return sanitizeOperateLogRawMessageWithFields(raw, nil)
}

func sanitizeOperateLogRawMessageWithFields(raw json.RawMessage, sensitiveFields map[string]struct{}) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	var value interface{}
	if err := json.Unmarshal(raw, &value); err != nil {
		return raw
	}
	redacted := newOperateLogRedactor(sensitiveFields).redact(value)
	if redacted == nil {
		return nil
	}
	data, err := json.Marshal(redacted)
	if err != nil {
		return nil
	}
	return data
}

func redactOperateLogValue(value interface{}) interface{} {
	return redactOperateLogValueWithFields(value, nil)
}

func redactOperateLogValueWithFields(value interface{}, sensitiveFields map[string]struct{}) interface{} {
	return newOperateLogRedactor(sensitiveFields).redact(value)
}

func (r operateLogRedactor) redact(value interface{}) interface{} {
	result := r.redactAtPath(value, "")
	if result.removed {
		return nil
	}
	return result.value
}

func (r operateLogRedactor) redactAtPath(value interface{}, currentPath string) operateLogRedactionResult {
	if r.isSensitiveFieldPath(currentPath) {
		return operateLogRedactionResult{removed: true}
	}
	switch typed := value.(type) {
	case nil:
		return operateLogRedactionResult{value: nil}
	case string, bool, float64, float32, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return operateLogRedactionResult{value: typed}
	case json.RawMessage:
		var decoded interface{}
		if err := json.Unmarshal(typed, &decoded); err != nil {
			return operateLogRedactionResult{value: string(typed)}
		}
		return r.redactAtPath(decoded, currentPath)
	case map[string]interface{}:
		redacted := make(map[string]interface{}, len(typed))
		for key, item := range typed {
			childPath := joinOperateLogFieldPath(currentPath, key)
			if isSensitiveOperateLogKey(key) || r.isSensitiveFieldPath(childPath) {
				continue
			}
			result := r.redactAtPath(item, childPath)
			if result.removed {
				continue
			}
			redacted[key] = result.value
		}
		return operateLogRedactionResult{value: redacted}
	case []interface{}:
		redacted := make([]interface{}, 0, len(typed))
		for _, item := range typed {
			result := r.redactAtPath(item, currentPath)
			if result.removed {
				continue
			}
			redacted = append(redacted, result.value)
		}
		return operateLogRedactionResult{value: redacted}
	default:
		data, err := json.Marshal(typed)
		if err != nil {
			return operateLogRedactionResult{value: typed}
		}
		var decoded interface{}
		if err := json.Unmarshal(data, &decoded); err != nil {
			return operateLogRedactionResult{value: typed}
		}
		return r.redactAtPath(decoded, currentPath)
	}
}

func isSensitiveOperateLogKey(key string) bool {
	normalized := strings.TrimSpace(key)
	if normalized == "" {
		return false
	}
	return sensitiveOperateLogKeyPattern.MatchString(normalized)
}

func (r operateLogRedactor) isSensitiveFieldPath(fieldPath string) bool {
	if len(r.sensitiveFields) == 0 || strings.TrimSpace(fieldPath) == "" {
		return false
	}
	_, ok := r.sensitiveFields[fieldPath]
	return ok
}

func joinOperateLogFieldPath(prefix, key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return strings.TrimSpace(prefix)
	}
	if strings.TrimSpace(prefix) == "" {
		return key
	}
	return strings.TrimSpace(prefix) + "." + key
}
