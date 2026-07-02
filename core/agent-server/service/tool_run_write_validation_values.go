package service

import (
	"encoding/json"
	"strings"

	"github.com/kageos/kageos-sdk/agent-app/widget"
)

func isRunWriteRequiredField(field *widget.Field) bool {
	for _, part := range strings.Split(field.Validation, ",") {
		part = strings.TrimSpace(part)
		if part == "required" {
			return true
		}
	}
	return false
}

func isRunWriteEmptyValue(value interface{}) bool {
	if value == nil {
		return true
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v) == ""
	case []interface{}:
		if len(v) == 0 {
			return true
		}
		values, invalid := runWriteStringValues(v)
		return !invalid && len(values) == 0
	case []string:
		return len(cleanRunWriteStrings(v)) == 0
	default:
		return false
	}
}

func runWriteFieldPath(label string, code string) string {
	if strings.TrimSpace(label) == "" {
		return code
	}
	return strings.TrimSpace(label) + "." + code
}

func runWriteFieldDisplayName(field *widget.Field) string {
	if field == nil {
		return "字段"
	}
	if strings.TrimSpace(field.Name) != "" {
		return field.Name
	}
	if strings.TrimSpace(field.FieldName) != "" {
		return field.FieldName
	}
	if strings.TrimSpace(field.Code) != "" {
		return field.Code
	}
	return "字段"
}

func runWriteConfigOptions(config interface{}) []string {
	m := runWriteConfigMap(config)
	if len(m) == 0 {
		return nil
	}
	return runWriteInterfaceStrings(m["options"])
}

func runWriteConfigBool(config interface{}, key string) bool {
	m := runWriteConfigMap(config)
	if len(m) == 0 {
		return false
	}
	switch v := m[key].(type) {
	case bool:
		return v
	case string:
		return strings.EqualFold(strings.TrimSpace(v), "true")
	default:
		return false
	}
}

func runWriteConfigMap(config interface{}) map[string]interface{} {
	if config == nil {
		return nil
	}
	if m, ok := config.(map[string]interface{}); ok {
		return m
	}
	data, err := json.Marshal(config)
	if err != nil {
		return nil
	}
	var out map[string]interface{}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil
	}
	return out
}

func runWriteInterfaceStrings(value interface{}) []string {
	switch v := value.(type) {
	case []string:
		return cleanRunWriteStrings(v)
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return cleanRunWriteStrings(out)
	case string:
		return splitRunWriteCSV(v)
	default:
		return nil
	}
}
