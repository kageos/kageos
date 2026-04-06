package v1

import (
	"encoding/json"
	"testing"

	"github.com/ai-agent-os/ai-agent-os/dto"
)

func TestBuildFormOperateLogResponseBody(t *testing.T) {
	resp := &dto.RequestAppResp{
		TraceId: "trace-1",
		Version: "v2",
		Result: map[string]interface{}{
			"id": 1,
		},
	}

	body := buildFormOperateLogResponseBody(resp, nil, 128)
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("unmarshal body failed: %v", err)
	}

	if payload["code"] != float64(0) {
		t.Fatalf("code = %v, want 0", payload["code"])
	}
	if payload["trace_id"] != "trace-1" {
		t.Fatalf("trace_id = %v, want trace-1", payload["trace_id"])
	}
	if payload["version"] != "v2" {
		t.Fatalf("version = %v, want v2", payload["version"])
	}
	if payload["total_cost_mill"] != float64(128) {
		t.Fatalf("total_cost_mill = %v, want 128", payload["total_cost_mill"])
	}
	if _, ok := payload["result"].(map[string]interface{}); !ok {
		t.Fatalf("result missing or invalid: %v", payload["result"])
	}
}

func TestBuildFormOperateLogResponseBodyWithBusinessError(t *testing.T) {
	resp := &dto.RequestAppResp{
		ErrCode: -1,
		Error:   "参数校验失败",
	}

	body := buildFormOperateLogResponseBody(resp, nil, 35)
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("unmarshal body failed: %v", err)
	}

	if payload["code"] != float64(-1) {
		t.Fatalf("code = %v, want -1", payload["code"])
	}
	if payload["msg"] != "参数校验失败" {
		t.Fatalf("msg = %v, want 参数校验失败", payload["msg"])
	}
	if payload["total_cost_mill"] != float64(35) {
		t.Fatalf("total_cost_mill = %v, want 35", payload["total_cost_mill"])
	}
}
