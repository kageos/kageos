package v1

import (
	"encoding/json"
	"testing"
)

func TestMergePublicSharePresetValuesOverridesSubmittedFields(t *testing.T) {
	body := []byte(`{"topic_id":2,"option_ids":[7],"remark":"keep"}`)
	preset := json.RawMessage(`{"topic_id":1}`)

	merged, err := mergePublicSharePresetValues(body, preset)
	if err != nil {
		t.Fatalf("mergePublicSharePresetValues() error = %v", err)
	}

	var got map[string]interface{}
	if err := json.Unmarshal(merged, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v; body = %s", err, merged)
	}
	if got["topic_id"] != float64(1) {
		t.Fatalf("topic_id = %#v, want 1", got["topic_id"])
	}
	if got["remark"] != "keep" {
		t.Fatalf("remark = %#v, want keep", got["remark"])
	}
	if _, ok := got["option_ids"].([]interface{}); !ok {
		t.Fatalf("option_ids missing after merge: %#v", got)
	}
}

func TestMergePublicSharePresetValuesIntoCallbackRequestOverridesCurrentFormData(t *testing.T) {
	body := []byte(`{"code":"option_ids","type":"by_keyword","value":"","request":{"topic_id":2,"remark":"keep"},"value_type":"[]int"}`)
	preset := json.RawMessage(`{"topic_id":1}`)

	merged, err := mergePublicSharePresetValuesIntoCallbackRequest(body, preset)
	if err != nil {
		t.Fatalf("mergePublicSharePresetValuesIntoCallbackRequest() error = %v", err)
	}

	var got struct {
		Code    string                 `json:"code"`
		Request map[string]interface{} `json:"request"`
	}
	if err := json.Unmarshal(merged, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v; body = %s", err, merged)
	}
	if got.Code != "option_ids" {
		t.Fatalf("code = %q, want option_ids", got.Code)
	}
	if got.Request["topic_id"] != float64(1) {
		t.Fatalf("request.topic_id = %#v, want 1", got.Request["topic_id"])
	}
	if got.Request["remark"] != "keep" {
		t.Fatalf("request.remark = %#v, want keep", got.Request["remark"])
	}
}

func TestMergePublicSharePresetValuesRejectsNonObjectSubmitBody(t *testing.T) {
	_, err := mergePublicSharePresetValues([]byte(`[]`), json.RawMessage(`{"topic_id":1}`))
	if err == nil {
		t.Fatal("expected non-object body error")
	}
}
