package service

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
)

func parseScheduledPayload(raw string) (interface{}, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return map[string]interface{}{}, nil
	}
	var out interface{}
	if err := json.Unmarshal([]byte(value), &out); err != nil {
		return nil, err
	}
	if out == nil {
		return map[string]interface{}{}, nil
	}
	return out, nil
}

func scheduledBodyFromCompatValue(value interface{}) string {
	raw := scheduledRawJSONFromCompatValue(value)
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	return unwrapScheduledInvokeParamsBody(raw)
}

func scheduledRawJSONFromCompatValue(value interface{}) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(data))
	}
}

func unwrapScheduledInvokeParamsBody(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	var wrapper map[string]interface{}
	if err := json.Unmarshal([]byte(value), &wrapper); err != nil {
		return value
	}
	for _, key := range []string{"body", "payload"} {
		if nested, ok := wrapper[key]; ok {
			nestedRaw := scheduledRawJSONFromCompatValue(nested)
			if strings.TrimSpace(nestedRaw) != "" {
				return nestedRaw
			}
		}
	}
	return value
}

func parseScheduledCompatInt(value interface{}) int {
	normalize := func(n int) int {
		if n > 0 {
			return n
		}
		return 0
	}
	switch v := value.(type) {
	case nil:
		return 0
	case int:
		return normalize(v)
	case int64:
		return normalize(int(v))
	case float64:
		return normalize(int(v))
	case json.Number:
		n, _ := v.Int64()
		return normalize(int(n))
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err == nil {
			return normalize(n)
		}
	}
	return 0
}

func mustRawJSON(value interface{}) json.RawMessage {
	data, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return data
}

func scheduledIdempotencyKey(explicit string, parts ...interface{}) string {
	if key := strings.TrimSpace(explicit); key != "" {
		return key
	}
	data, _ := json.Marshal(parts)
	sum := sha1.Sum(data)
	return "agent-scheduled-" + hex.EncodeToString(sum[:])[:24]
}
