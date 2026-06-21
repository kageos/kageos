package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/pkg/functionschema"
	"github.com/kageos/kageos-sdk/agent-app/widget"
)

func TestConvertFunctionToRespNormalizesStoredFieldCodes(t *testing.T) {
	svc := &FunctionService{}
	schemaJSON, err := functionschema.Marshal(functionschema.NewForm(
		[]*widget.Field{
			{
				Code:      "input_files,omitempty",
				FieldName: "InputFiles",
				Name:      "输入文件",
				Widget: struct {
					Type   string      `json:"type"`
					Config interface{} `json:"config,omitempty"`
				}{Type: "text_area", Config: map[string]interface{}{}},
			},
		},
		[]*widget.Field{
			{
				Code:      "output_files,omitempty",
				FieldName: "OutputFiles",
				Name:      "输出文件",
				Widget: struct {
					Type   string      `json:"type"`
					Config interface{} `json:"config,omitempty"`
				}{Type: "files", Config: map[string]interface{}{"max_count": 5}},
			},
		},
		nil,
	))
	if err != nil {
		t.Fatalf("marshal schema failed: %v", err)
	}
	function := &model.Function{
		Schema: schemaJSON,
	}

	resp := svc.convertFunctionToResp(context.Background(), function)

	requestFields := resp.Schema.Form.Request
	responseFields := resp.Schema.Form.Response

	requestJSON, err := json.Marshal(requestFields)
	if err != nil {
		t.Fatalf("marshal request failed: %v", err)
	}
	responseJSON, err := json.Marshal(responseFields)
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

	if got := requestFieldsData[0]["code"]; got != "input_files" {
		t.Fatalf("request code = %#v, want %q", got, "input_files")
	}
	if got := responseFieldsData[0]["code"]; got != "output_files" {
		t.Fatalf("response code = %#v, want %q", got, "output_files")
	}
	if requestFields[0].Code != "input_files" {
		t.Fatalf("requestFields[0].Code = %q, want %q", requestFields[0].Code, "input_files")
	}
	if responseFields[0].Code != "output_files" {
		t.Fatalf("responseFields[0].Code = %q, want %q", responseFields[0].Code, "output_files")
	}
}
