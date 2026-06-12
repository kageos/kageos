package service

import (
	"strings"
	"testing"
)

func TestValidateToolArgumentsRejectsMissingRequired(t *testing.T) {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{"type": "string"},
		},
		"required": []interface{}{"name"},
	}

	err := validateToolArguments(schema, map[string]interface{}{})
	if err == nil || !strings.Contains(err.Error(), "arguments.name 缺少必填参数") {
		t.Fatalf("validateToolArguments() error = %v, want missing required", err)
	}
}

func TestValidateToolArgumentsRejectsTypeAndEnum(t *testing.T) {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"page":   map[string]interface{}{"type": "integer"},
			"format": map[string]interface{}{"type": "string", "enum": []interface{}{"tree", "list"}},
		},
	}

	err := validateToolArguments(schema, map[string]interface{}{
		"page":   "one",
		"format": "grid",
	})
	if err == nil {
		t.Fatal("validateToolArguments() error = nil, want validation error")
	}
	if !strings.Contains(err.Error(), "arguments.page 类型错误") || !strings.Contains(err.Error(), "arguments.format 值") {
		t.Fatalf("validateToolArguments() error = %v, want type and enum errors", err)
	}
}

func TestNormalizeToolArgumentsForSchemaCoercesNumericStrings(t *testing.T) {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"page":       map[string]interface{}{"type": "integer"},
			"confidence": map[string]interface{}{"type": "number"},
			"payload": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"ids": map[string]interface{}{
						"type":  "array",
						"items": map[string]interface{}{"type": "integer"},
					},
				},
			},
		},
	}

	got := normalizeToolArgumentsForSchema(schema, map[string]interface{}{
		"page":       "3",
		"confidence": "0.75",
		"payload": map[string]interface{}{
			"ids": []interface{}{"1", "2"},
		},
	})
	if err := validateToolArguments(schema, got); err != nil {
		t.Fatalf("normalized args should validate: %v", err)
	}
	if got["page"] != int64(3) {
		t.Fatalf("page=%#v, want int64(3)", got["page"])
	}
	if got["confidence"] != 0.75 {
		t.Fatalf("confidence=%#v, want 0.75", got["confidence"])
	}
	payload := got["payload"].(map[string]interface{})
	ids := payload["ids"].([]interface{})
	if ids[0] != int64(1) || ids[1] != int64(2) {
		t.Fatalf("ids=%#v, want normalized integer values", ids)
	}
}

func TestValidateToolArgumentsAllowsNestedValidValues(t *testing.T) {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"payload": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"ids": map[string]interface{}{
						"type":  "array",
						"items": map[string]interface{}{"type": "integer"},
					},
				},
			},
		},
	}

	err := validateToolArguments(schema, map[string]interface{}{
		"payload": map[string]interface{}{
			"ids": []interface{}{float64(1), 2},
		},
	})
	if err != nil {
		t.Fatalf("validateToolArguments() error = %v, want nil", err)
	}
}

func TestValidateToolArgumentsAllowsTypedSlicesAndMaps(t *testing.T) {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"rules": map[string]interface{}{
				"type":  "array",
				"items": map[string]interface{}{"type": "string"},
			},
			"rows": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"name": map[string]interface{}{"type": "string"},
					},
					"required": []interface{}{"name"},
				},
			},
		},
	}

	err := validateToolArguments(schema, map[string]interface{}{
		"rules": []string{"先读 schema", "再执行"},
		"rows":  []map[string]interface{}{{"name": "NPS"}},
	})
	if err != nil {
		t.Fatalf("validateToolArguments() error = %v, want nil", err)
	}
}
