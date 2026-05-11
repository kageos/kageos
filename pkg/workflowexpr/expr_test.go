package workflowexpr

import (
	"encoding/json"
	"testing"
)

func TestResolveRefAndConst(t *testing.T) {
	t.Parallel()

	ctx := Context{
		"input": map[string]interface{}{
			"title": "季度报告",
		},
		"steps": map[string]interface{}{
			"extract": map[string]interface{}{
				"output": map[string]interface{}{
					"text": "hello",
				},
			},
		},
	}

	value, err := ResolveRaw(json.RawMessage(`{"$ref":"steps.extract.output.text"}`), ctx)
	if err != nil {
		t.Fatal(err)
	}
	if value != "hello" {
		t.Fatalf("value = %#v, want hello", value)
	}

	value, err = ResolveRaw(json.RawMessage(`{"$const":"正式"}`), ctx)
	if err != nil {
		t.Fatal(err)
	}
	if value != "正式" {
		t.Fatalf("value = %#v, want 正式", value)
	}
}

func TestValidateRejectsUnsupportedOperator(t *testing.T) {
	t.Parallel()

	err := ValidateRaw(json.RawMessage(`{"$fn":"concat","args":[]}`), MVPOptions())
	if err == nil {
		t.Fatal("expected unsupported operator error")
	}
}

func TestLookupArrayIndex(t *testing.T) {
	t.Parallel()

	value, err := Lookup(Context{
		"steps": map[string]interface{}{
			"a": map[string]interface{}{
				"output": map[string]interface{}{
					"items": []interface{}{"first", "second"},
				},
			},
		},
	}, "steps.a.output.items.1")
	if err != nil {
		t.Fatal(err)
	}
	if value != "second" {
		t.Fatalf("value = %#v, want second", value)
	}
}
