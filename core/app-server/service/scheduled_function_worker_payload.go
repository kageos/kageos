package service

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/kageos/kageos/pkg/access"
	"github.com/kageos/kageos/pkg/scheduledsdk"
)

func decodeScheduledFunctionPayload(event scheduledsdk.ExecutionRequestedEvent) (scheduledFunctionPayload, error) {
	var payload scheduledFunctionPayload
	if len(event.ExecutorPayload) == 0 {
		return payload, fmt.Errorf("scheduled function executor_payload is empty")
	}
	if err := json.Unmarshal(event.ExecutorPayload, &payload); err != nil {
		return payload, fmt.Errorf("decode scheduled function payload: %w", err)
	}
	payload.FullCodePath = access.NormalizeResourcePath(payload.FullCodePath)
	payload.TemplateType = strings.TrimSpace(payload.TemplateType)
	payload.Action = strings.TrimSpace(payload.Action)
	if payload.Action == "" {
		payload.Action = "execute"
	}
	payload.Method = strings.TrimSpace(payload.Method)
	if len(payload.Payload) == 0 && len(payload.Body) > 0 {
		payload.Payload = payload.Body
	}
	if len(payload.Payload) == 0 {
		payload.Payload = json.RawMessage(`{}`)
	}
	if payload.FullCodePath == "" {
		return payload, fmt.Errorf("scheduled function payload requires full_code_path")
	}
	if payload.TemplateType == "" {
		payload.TemplateType = scheduledFunctionTemplateType(payload.FullCodePath)
	}
	if !isScheduledFunctionAction(payload.Action) {
		return payload, fmt.Errorf("scheduled function action %q is not supported", payload.Action)
	}
	return payload, nil
}

func scheduledFunctionPayloadBodyBytes(raw json.RawMessage) []byte {
	raw = compactScheduledRawJSON(raw)
	if len(raw) == 0 {
		return []byte("{}")
	}
	return []byte(raw)
}

func scheduledFunctionPayloadURLQuery(raw json.RawMessage) (string, error) {
	raw = compactScheduledRawJSON(raw)
	if len(raw) == 0 || string(raw) == "{}" {
		return "", nil
	}
	var queryString string
	if err := json.Unmarshal(raw, &queryString); err == nil {
		return strings.TrimPrefix(strings.TrimSpace(queryString), "?"), nil
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", fmt.Errorf("scheduled function query payload must be JSON object or query string: %w", err)
	}
	values := url.Values{}
	for key, value := range payload {
		key = strings.TrimSpace(key)
		if key == "" || value == nil {
			continue
		}
		switch typed := value.(type) {
		case []interface{}:
			for _, item := range typed {
				if item != nil {
					values.Add(key, fmt.Sprint(item))
				}
			}
		default:
			values.Set(key, fmt.Sprint(typed))
		}
	}
	return values.Encode(), nil
}

func compactScheduledRawJSON(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	var out interface{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return raw
	}
	if out == nil {
		return nil
	}
	data, err := json.Marshal(out)
	if err != nil {
		return raw
	}
	return data
}

func needFillScheduledOldValues(bodyData map[string]interface{}) bool {
	if bodyData == nil {
		return false
	}
	if _, hasID := bodyData["id"]; !hasID {
		return false
	}
	oldValues, ok := bodyData["old_values"].(map[string]interface{})
	return !ok || len(oldValues) == 0
}

func getScheduledBodyIDInt64(bodyData map[string]interface{}) (int64, bool) {
	if bodyData == nil {
		return 0, false
	}
	switch v := bodyData["id"].(type) {
	case float64:
		return int64(v), true
	case int:
		return int64(v), true
	case int64:
		return v, true
	default:
		return 0, false
	}
}

func scheduledTableRowID(row map[string]interface{}) (int64, bool) {
	if row == nil {
		return 0, false
	}
	switch v := row["id"].(type) {
	case float64:
		return int64(v), true
	case int:
		return int64(v), true
	case int64:
		return v, true
	case json.Number:
		id, err := v.Int64()
		return id, err == nil
	case string:
		id, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		return id, err == nil
	default:
		return 0, false
	}
}

func findScheduledTableRowByID(rows []map[string]interface{}, id int64) map[string]interface{} {
	for _, row := range rows {
		rowID, ok := scheduledTableRowID(row)
		if ok && rowID == id {
			return row
		}
	}
	if len(rows) == 1 {
		return rows[0]
	}
	return nil
}

func extractScheduledTableGetRowsCallbackRows(result interface{}) ([]map[string]interface{}, error) {
	data, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("解析 %s 结果失败: %w", internalTableGetRowsCallback, err)
	}
	var payload struct {
		Rows []map[string]interface{} `json:"rows"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("解析 %s rows 失败: %w", internalTableGetRowsCallback, err)
	}
	if payload.Rows == nil {
		return nil, fmt.Errorf("%s 未返回 rows", internalTableGetRowsCallback)
	}
	return payload.Rows, nil
}

func scheduledRowIDsFromDeleteBody(bodyData map[string]interface{}) []int64 {
	ids, ok := bodyData["ids"].([]interface{})
	if !ok {
		return nil
	}
	rowIDs := make([]int64, 0, len(ids))
	for _, id := range ids {
		switch v := id.(type) {
		case float64:
			rowIDs = append(rowIDs, int64(v))
		case int:
			rowIDs = append(rowIDs, int64(v))
		case int64:
			rowIDs = append(rowIDs, v)
		case string:
			parsed, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
			if err == nil {
				rowIDs = append(rowIDs, parsed)
			}
		}
	}
	return rowIDs
}
