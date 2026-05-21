package v1

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestBuildCallbackAppReqEncodesBodyForSDKCallbackRouter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	body := `{"code":"topic_id","type":"by_keyword","value":"","request":{"topic_id":null,"option_ids":[],"remark":""},"value_type":"int"}`
	req := httptest.NewRequest(http.MethodPost, "/workspace/api/v1/callback/on_select_fuzzy/liubeiluo/ee/vote/vote_submit.form", strings.NewReader(body))
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req

	api := &StandardAPI{}
	appReq, err := api.buildCallbackAppReq(ctx, "/liubeiluo/ee/vote/vote_submit.form", "OnSelectFuzzy")
	if err != nil {
		t.Fatalf("buildCallbackAppReq() error = %v", err)
	}

	var sdkReq struct {
		Type   string `json:"type"`
		Method string `json:"method"`
		Router string `json:"router"`
		Body   []byte `json:"body"`
	}
	if err := json.Unmarshal(appReq.Body, &sdkReq); err != nil {
		t.Fatalf("json.Unmarshal(appReq.Body) error = %v; body = %s", err, string(appReq.Body))
	}

	if sdkReq.Type != "OnSelectFuzzy" {
		t.Fatalf("Type = %q, want OnSelectFuzzy", sdkReq.Type)
	}
	if sdkReq.Method != http.MethodPost {
		t.Fatalf("Method = %q, want POST", sdkReq.Method)
	}
	if sdkReq.Router != "vote/vote_submit.form" {
		t.Fatalf("Router = %q, want vote/vote_submit.form", sdkReq.Router)
	}
	if string(sdkReq.Body) != body {
		t.Fatalf("Body = %s, want %s", string(sdkReq.Body), body)
	}
}
