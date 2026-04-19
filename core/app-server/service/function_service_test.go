package service

import (
	"encoding/json"
	"testing"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/widget"
)

func TestConvertFunctionToRespNormalizesStoredFieldCodes(t *testing.T) {
	svc := &FunctionService{}
	function := &model.Function{
		Request: json.RawMessage(`[
			{"code":"args_json,omitempty","field_name":"ArgsJSON","name":"参数(JSON格式)","widget":{"type":"text_area","config":{}}}
		]`),
		Response: json.RawMessage(`[
			{"code":"output_files,omitempty","field_name":"OutputFiles","name":"输出文件","widget":{"type":"files","config":{"max_count":5}}}
		]`),
	}

	resp := svc.convertFunctionToResp(function)

	requestFields, ok := resp.Request.([]*widget.Field)
	if !ok {
		t.Fatalf("request type = %T, want []*widget.Field", resp.Request)
	}
	responseFields, ok := resp.Response.([]*widget.Field)
	if !ok {
		t.Fatalf("response type = %T, want []*widget.Field", resp.Response)
	}

	requestJSON, err := json.Marshal(resp.Request)
	if err != nil {
		t.Fatalf("marshal request failed: %v", err)
	}
	responseJSON, err := json.Marshal(resp.Response)
	if err != nil {
		t.Fatalf("marshal response failed: %v", err)
	}

	var requestFieldsData []map[string]any
	if err := json.Unmarshal(requestJSON, &requestFieldsData); err != nil {
		t.Fatalf("unmarshal normalized request failed: %v", err)
	}
	var responseFieldsData []map[string]any
	if err := json.Unmarshal(responseJSON, &responseFieldsData); err != nil {
		t.Fatalf("unmarshal normalized response failed: %v", err)
	}

	if got := requestFieldsData[0]["code"]; got != "args_json" {
		t.Fatalf("request code = %#v, want %q", got, "args_json")
	}
	if got := responseFieldsData[0]["code"]; got != "output_files" {
		t.Fatalf("response code = %#v, want %q", got, "output_files")
	}
	if requestFields[0].Code != "args_json" {
		t.Fatalf("requestFields[0].Code = %q, want %q", requestFields[0].Code, "args_json")
	}
	if responseFields[0].Code != "output_files" {
		t.Fatalf("responseFields[0].Code = %q, want %q", responseFields[0].Code, "output_files")
	}
}
