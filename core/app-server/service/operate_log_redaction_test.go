package service

import (
	"encoding/json"
	"testing"
)

func TestSanitizeOperateLogRawMessageRedactsSensitiveFields(t *testing.T) {
	raw := json.RawMessage(`{
		"username": "alice",
		"password": "plain-password",
		"profile": {
			"api_key": "secret-api-key",
			"nickname": "Alice"
		},
		"items": [
			{"token": "row-token", "name": "visible"}
		]
	}`)

	sanitized := sanitizeOperateLogRawMessage(raw)
	var payload map[string]interface{}
	if err := json.Unmarshal(sanitized, &payload); err != nil {
		t.Fatalf("unmarshal sanitized payload: %v", err)
	}

	if payload["username"] != "alice" {
		t.Fatalf("username should stay visible: %+v", payload)
	}
	if _, exists := payload["password"]; exists {
		t.Fatalf("password should be removed from audit payload: %+v", payload)
	}

	profile := payload["profile"].(map[string]interface{})
	if _, exists := profile["api_key"]; exists {
		t.Fatalf("api_key should be removed from nested audit payload: %+v", profile)
	}
	if profile["nickname"] != "Alice" {
		t.Fatalf("nested profile was not redacted correctly: %+v", profile)
	}

	items := payload["items"].([]interface{})
	first := items[0].(map[string]interface{})
	if _, exists := first["token"]; exists {
		t.Fatalf("token should be removed from array audit payload: %+v", first)
	}
	if first["name"] != "visible" {
		t.Fatalf("array item was not redacted correctly: %+v", first)
	}
}

func TestMustMarshalRawRedactsStructFields(t *testing.T) {
	type payload struct {
		Name   string `json:"name"`
		Secret string `json:"secret"`
	}

	raw := mustMarshalRaw(payload{Name: "public", Secret: "private"})
	var decoded map[string]interface{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}
	if decoded["name"] != "public" {
		t.Fatalf("name should stay visible: %+v", decoded)
	}
	if _, exists := decoded["secret"]; exists {
		t.Fatalf("secret should be removed from audit payload: %+v", decoded)
	}
}

func TestSanitizeOperateLogRawMessageRedactsSchemaFieldPaths(t *testing.T) {
	raw := json.RawMessage(`{
		"member_id": 1,
		"customer": {
			"name": "Alice",
			"card_number": "VIP-001"
		},
		"items": [
			{"product_id": 10, "serial": "S-1"}
		]
	}`)
	fields := map[string]struct{}{
		"customer.card_number": {},
		"items.serial":         {},
	}

	sanitized := sanitizeOperateLogRawMessageWithFields(raw, fields)
	var payload map[string]interface{}
	if err := json.Unmarshal(sanitized, &payload); err != nil {
		t.Fatalf("unmarshal sanitized payload: %v", err)
	}

	customer := payload["customer"].(map[string]interface{})
	if _, exists := customer["card_number"]; exists {
		t.Fatalf("schema sensitive path should be removed: %+v", customer)
	}
	if customer["name"] != "Alice" {
		t.Fatalf("nested schema path was not redacted correctly: %+v", customer)
	}
	items := payload["items"].([]interface{})
	first := items[0].(map[string]interface{})
	if _, exists := first["serial"]; exists {
		t.Fatalf("array schema sensitive path should be removed: %+v", first)
	}
	if first["product_id"] != float64(10) {
		t.Fatalf("array schema path was not redacted correctly: %+v", first)
	}
}
