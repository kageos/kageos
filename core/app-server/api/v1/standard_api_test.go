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
	req := httptest.NewRequest(http.MethodPost, "/workspace/api/v1/selection-options/liubeiluo/ee/vote/vote_submit.form", strings.NewReader(body))
	req.Header.Set("X-Client-Source", "agent")
	req.Header.Set("X-Source-Type", "agent_tool")
	req.Header.Set("X-Source-Ref", "session-1")
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
	if appReq.TargetRouter != "vote/vote_submit.form" {
		t.Fatalf("TargetRouter = %q, want vote/vote_submit.form", appReq.TargetRouter)
	}
	if string(sdkReq.Body) != body {
		t.Fatalf("Body = %s, want %s", string(sdkReq.Body), body)
	}
	if appReq.ClientSource != "agent" || appReq.SourceType != "agent_tool" || appReq.SourceRef != "session-1" {
		t.Fatalf("source context mismatch: source=%q type=%q ref=%q", appReq.ClientSource, appReq.SourceType, appReq.SourceRef)
	}
}

func TestBuildCallbackAppReqWithBodyEncodesSystemTableGetRows(t *testing.T) {
	gin.SetMode(gin.TestMode)

	body := []byte(`{"ids":[7]}`)
	req := httptest.NewRequest(http.MethodPut, "/workspace/api/v1/tables/liubeiluo/ee/vote/vote_topic.table", nil)
	req.Header.Set("X-Client-Source", "agent")
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req

	api := &StandardAPI{}
	appReq, err := api.buildCallbackAppReqWithBody(ctx, "/liubeiluo/ee/vote/vote_topic.table", internalTableGetRowsCallback, tableGetRowsCallbackHTTPMethod, body, "")
	if err != nil {
		t.Fatalf("buildCallbackAppReqWithBody() error = %v", err)
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

	if sdkReq.Type != internalTableGetRowsCallback {
		t.Fatalf("Type = %q, want %s", sdkReq.Type, internalTableGetRowsCallback)
	}
	if sdkReq.Method != tableGetRowsCallbackHTTPMethod {
		t.Fatalf("Method = %q, want %s", sdkReq.Method, tableGetRowsCallbackHTTPMethod)
	}
	if sdkReq.Router != "vote/vote_topic.table" || appReq.TargetRouter != "vote/vote_topic.table" {
		t.Fatalf("router mismatch: envelope=%q target=%q", sdkReq.Router, appReq.TargetRouter)
	}
	if string(sdkReq.Body) != string(body) {
		t.Fatalf("Body = %s, want %s", string(sdkReq.Body), string(body))
	}
	if appReq.Router != "/_callback" {
		t.Fatalf("Router = %q, want /_callback", appReq.Router)
	}
}

func TestExtractTableGetRowsCallbackRowsFindsMatchingID(t *testing.T) {
	result := map[string]interface{}{
		"rows": []interface{}{
			map[string]interface{}{"id": float64(1), "name": "first"},
			map[string]interface{}{"id": float64(7), "name": "target"},
		},
	}
	rows, err := extractTableGetRowsCallbackRows(result)
	if err != nil {
		t.Fatalf("extractTableGetRowsCallbackRows() error = %v, want nil", err)
	}
	row := findTableRowByID(rows, 7)
	if row == nil || row["name"] != "target" {
		t.Fatalf("findTableRowByID() = %#v, want target row", row)
	}
}

func TestBuildRuntimePythonRequestAppReqUsesPrivateRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)

	body := `{"python_code":"def kageos_entry(args, output_dir):\n    return {\"data\":{\"ok\": True}}","args":{"name":"demo"},"timeout_seconds":30,"collect_output_files":true}`
	req := httptest.NewRequest(http.MethodPost, "/workspace/api/v1/python-executions/liubeiluo/ee/vote/vote_submit.form", strings.NewReader(body))
	req.Header.Set("X-Client-Source", "agent")
	req.Header.Set("X-Source-Type", "agent_tool")
	req.Header.Set("X-Source-Ref", "session-1")
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = req

	api := &StandardAPI{}
	appReq, err := api.buildRuntimePythonRequestAppReq(ctx, "/liubeiluo/ee/vote/vote_submit.form")
	if err != nil {
		t.Fatalf("buildRuntimePythonRequestAppReq() error = %v", err)
	}

	if appReq.User != "liubeiluo" || appReq.App != "ee" {
		t.Fatalf("target app mismatch: user=%q app=%q", appReq.User, appReq.App)
	}
	if appReq.Router != privateRuntimePythonRouter {
		t.Fatalf("Router = %q, want %q", appReq.Router, privateRuntimePythonRouter)
	}
	if appReq.Method != http.MethodPost {
		t.Fatalf("Method = %q, want POST", appReq.Method)
	}
	if appReq.ClientSource != "agent" || appReq.SourceType != "agent_tool" || appReq.SourceRef != "session-1" {
		t.Fatalf("source context mismatch: source=%q type=%q ref=%q", appReq.ClientSource, appReq.SourceType, appReq.SourceRef)
	}

	var runtimeReq struct {
		PythonCode         string                 `json:"python_code"`
		Args               map[string]interface{} `json:"args"`
		TimeoutSeconds     int                    `json:"timeout_seconds"`
		CollectOutputFiles bool                   `json:"collect_output_files"`
	}
	if err := json.Unmarshal(appReq.Body, &runtimeReq); err != nil {
		t.Fatalf("json.Unmarshal(appReq.Body) error = %v; body = %s", err, string(appReq.Body))
	}
	if !strings.Contains(runtimeReq.PythonCode, "kageos_entry") || runtimeReq.Args["name"] != "demo" {
		t.Fatalf("unexpected runtime body: %#v", runtimeReq)
	}
	if runtimeReq.TimeoutSeconds != 30 || !runtimeReq.CollectOutputFiles {
		t.Fatalf("runtime options mismatch: %#v", runtimeReq)
	}
}
