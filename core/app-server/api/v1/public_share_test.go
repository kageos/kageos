package v1

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
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

func TestPositivePublicQueryInt(t *testing.T) {
	for _, test := range []struct {
		raw      string
		fallback int
		want     int
	}{
		{raw: "2", fallback: 1, want: 2},
		{raw: "", fallback: 20, want: 20},
		{raw: "0", fallback: 20, want: 20},
		{raw: "invalid", fallback: 20, want: 20},
	} {
		if got := positivePublicQueryInt(test.raw, test.fallback); got != test.want {
			t.Fatalf("positivePublicQueryInt(%q, %d) = %d, want %d", test.raw, test.fallback, got, test.want)
		}
	}
}

func TestReadPublicShareRequestBodyRejectsOversizedChunkedBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	request := httptest.NewRequest(
		http.MethodPost,
		"/public/api/v1/shares/test/submit",
		bytes.NewReader(bytes.Repeat([]byte("x"), int(maxPublicShareRequestBodyBytes)+1)),
	)
	request.ContentLength = -1
	ctx.Request = request

	if _, err := readPublicShareRequestBody(ctx); err == nil {
		t.Fatal("oversized request body should be rejected")
	}
}

func TestReadPublicShareRequestBodyAcceptsBoundedBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"name":"Alice"}`))

	body, err := readPublicShareRequestBody(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `{"name":"Alice"}` {
		t.Fatalf("body = %q", body)
	}
}
