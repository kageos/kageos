package v1

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/core/agent-server/model"
	"github.com/kageos/kageos/core/agent-server/repository"
	"github.com/kageos/kageos/core/agent-server/service"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newLLMHandlerTestService(t *testing.T) *service.LLMService {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.LLMConfig{}); err != nil {
		t.Fatalf("migrate llm configs: %v", err)
	}
	return service.NewLLMService(repository.NewLLMRepository(db))
}

func TestLLMListMarksCurrentAdmins(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := newLLMHandlerTestService(t)
	if err := svc.CreateLLMConfig(contextx.WithRequestUser(context.Background(), "alice"), &model.LLMConfig{
		Name:      "Editable",
		Provider:  model.LLMProviderOpenAI,
		Model:     "gpt-test",
		APIKey:    "secret",
		Admin:     "alice",
		Timeout:   30,
		MaxTokens: 1024,
	}); err != nil {
		t.Fatalf("CreateLLMConfig() error = %v", err)
	}

	router := gin.New()
	router.GET("/llm/list", NewLLM(svc).List)

	req := httptest.NewRequest(http.MethodGet, "/llm/list?scope=mine&page=1&page_size=20", nil)
	req.Header.Set(contextx.RequestUserHeader, "alice")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}

	var body struct {
		Code int `json:"code"`
		Data struct {
			Configs []dto.LLMInfo `json:"configs"`
			Total   int64         `json:"total"`
		} `json:"data"`
		Msg string `json:"msg"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v; body = %s", err, w.Body.String())
	}
	if body.Code != 0 {
		t.Fatalf("response code = %d msg = %q", body.Code, body.Msg)
	}
	if body.Data.Total != 1 || len(body.Data.Configs) != 1 {
		t.Fatalf("configs = total %d list %#v, want one config", body.Data.Total, body.Data.Configs)
	}
	got := body.Data.Configs[0]
	if !got.IsAdmin {
		t.Fatalf("IsAdmin = false, want true for current admin")
	}
	if !got.HasAPIKey || got.APIKey != "" {
		t.Fatalf("API key list exposure mismatch: has=%v key=%q", got.HasAPIKey, got.APIKey)
	}
}

func TestLLMGetDoesNotExposeAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := newLLMHandlerTestService(t)
	cfg := &model.LLMConfig{
		Name:       "Private",
		Provider:   model.LLMProviderOpenAI,
		Model:      "gpt-test",
		APIKey:     "secret",
		Admin:      "alice",
		Timeout:    30,
		MaxTokens:  1024,
		Visibility: 1,
	}
	if err := svc.CreateLLMConfig(contextx.WithRequestUser(context.Background(), "alice"), cfg); err != nil {
		t.Fatalf("CreateLLMConfig() error = %v", err)
	}

	router := gin.New()
	router.GET("/llm/get", NewLLM(svc).Get)

	req := httptest.NewRequest(http.MethodGet, "/llm/get?id="+strconv.FormatInt(cfg.ID, 10), nil)
	req.Header.Set(contextx.RequestUserHeader, "alice")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}

	var body struct {
		Code int         `json:"code"`
		Data dto.LLMInfo `json:"data"`
		Msg  string      `json:"msg"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v; body = %s", err, w.Body.String())
	}
	if body.Code != 0 {
		t.Fatalf("response code = %d msg = %q", body.Code, body.Msg)
	}
	if !body.Data.HasAPIKey || body.Data.APIKey != "" {
		t.Fatalf("API key detail exposure mismatch: has=%v key=%q", body.Data.HasAPIKey, body.Data.APIKey)
	}
	if !body.Data.IsAdmin {
		t.Fatalf("IsAdmin = false, want true for current admin")
	}
}

func TestLLMInfoHidesEditableAdvancedFieldsFromNonAdmins(t *testing.T) {
	headers := `{"Authorization":"Bearer should-not-leak"}`
	extraConfig := `{"temperature":0.2}`
	capabilities := `{"stream":true}`
	info := llmInfoFromConfig(&model.LLMConfig{
		Name:         "Shared",
		Provider:     model.LLMProviderOpenAI,
		Model:        "gpt-test",
		Headers:      &headers,
		ExtraConfig:  &extraConfig,
		Capabilities: &capabilities,
		Admin:        "alice",
		Visibility:   0,
	}, "bob")

	if info.IsAdmin {
		t.Fatal("IsAdmin = true, want false")
	}
	if info.Headers != "" || info.ExtraConfig != "" || info.Admin != "" {
		t.Fatalf("editable advanced fields leaked: headers=%q extra_config=%q admin=%q", info.Headers, info.ExtraConfig, info.Admin)
	}
	if info.Capabilities == "" {
		t.Fatal("Capabilities should remain visible because it describes runtime support")
	}
}

func TestLLMGetDefaultReturnsNullWhenNotConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/llm/get_default", NewLLM(newLLMHandlerTestService(t)).GetDefault)

	req := httptest.NewRequest(http.MethodGet, "/llm/get_default", nil)
	req.Header.Set(contextx.RequestUserHeader, "alice")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}

	var body struct {
		Code int             `json:"code"`
		Data json.RawMessage `json:"data"`
		Msg  string          `json:"msg"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json.Unmarshal() error = %v; body = %s", err, w.Body.String())
	}
	if body.Code != 0 || string(body.Data) != "null" {
		t.Fatalf("response = code %d data %s msg %q, want successful null data", body.Code, body.Data, body.Msg)
	}
}

func TestLLMInfoInfersResponsesProtocolFromEndpoint(t *testing.T) {
	info := llmInfoFromConfig(&model.LLMConfig{
		Name:         "Responses",
		Provider:     model.LLMProviderOpenAI,
		Protocol:     model.LLMProtocolOpenAIChatCompletions,
		Model:        "gpt-test",
		APIBase:      "https://devcloud.chat",
		EndpointPath: "/responses",
	}, "alice")
	if info.Provider != model.LLMProviderOpenAI || info.Protocol != model.LLMProtocolOpenAIResponses {
		t.Fatalf("provider/protocol = %q/%q, want openai/openai_responses", info.Provider, info.Protocol)
	}
}
