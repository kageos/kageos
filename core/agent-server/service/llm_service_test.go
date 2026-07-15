package service

import (
	"context"
	"strings"
	"testing"

	"github.com/kageos/kageos/core/agent-server/model"
	"github.com/kageos/kageos/pkg/contextx"
	"gorm.io/gorm"
)

func TestLLMConfigAdminPermissionsAndNormalization(t *testing.T) {
	svc, repo := newLLMSeedTestService(t)
	aliceCtx := contextx.WithRequestUser(context.Background(), "alice")
	bobCtx := contextx.WithRequestUser(context.Background(), "bob")
	carolCtx := contextx.WithRequestUser(context.Background(), "carol")

	cfg := &model.LLMConfig{
		Name:      "Primary",
		Provider:  model.LLMProviderOpenAI,
		Model:     "gpt-test",
		APIKey:    "secret",
		Admin:     " alice, bob ,alice ",
		Timeout:   30,
		MaxTokens: 1024,
	}
	if err := svc.CreateLLMConfig(aliceCtx, cfg); err != nil {
		t.Fatalf("CreateLLMConfig() error = %v", err)
	}
	if cfg.APIKey == "secret" || !strings.HasPrefix(cfg.APIKey, llmAPIKeyCipherPrefix) {
		t.Fatalf("APIKey was not sealed: %q", cfg.APIKey)
	}
	opened, err := svc.OpenAPIKey(cfg.APIKey)
	if err != nil {
		t.Fatalf("OpenAPIKey(created) error = %v", err)
	}
	if opened != "secret" {
		t.Fatalf("OpenAPIKey(created) = %q, want original secret", opened)
	}
	if cfg.Admin != "alice,bob" {
		t.Fatalf("Admin = %q, want alice,bob", cfg.Admin)
	}
	if cfg.Provider != model.LLMProviderOpenAI || cfg.Protocol != model.LLMProtocolOpenAIChatCompletions {
		t.Fatalf("provider/protocol = %q/%q, want openai/openai_chat_completions", cfg.Provider, cfg.Protocol)
	}
	if !cfg.IsAdminUser("alice") || !cfg.IsAdminUser("bob") || cfg.IsAdminUser("carol") {
		t.Fatalf("IsAdminUser mismatch: admin=%q created_by=%q", cfg.Admin, cfg.CreatedBy)
	}

	list, total, err := svc.ListLLMConfigs(bobCtx, "mine", 1, 10)
	if err != nil {
		t.Fatalf("ListLLMConfigs() error = %v", err)
	}
	if total != 1 || len(list) != 1 || list[0].ID != cfg.ID {
		t.Fatalf("ListLLMConfigs() = total %d list %#v, want created config", total, list)
	}

	blockedUpdate := &model.LLMConfig{
		Name:      "Blocked",
		Provider:  model.LLMProviderOpenAI,
		Model:     "gpt-test",
		Timeout:   30,
		MaxTokens: 1024,
	}
	blockedUpdate.ID = cfg.ID
	err = svc.UpdateLLMConfig(carolCtx, blockedUpdate)
	if err == nil || !strings.Contains(err.Error(), "无权限修改") {
		t.Fatalf("UpdateLLMConfig() error = %v, want permission error", err)
	}

	allowedUpdate := &model.LLMConfig{
		Name:      "Allowed",
		Provider:  model.LLMProviderOpenAI,
		Model:     "gpt-test-new",
		APIKey:    "new-secret",
		Timeout:   60,
		MaxTokens: 2048,
	}
	allowedUpdate.ID = cfg.ID
	err = svc.UpdateLLMConfig(bobCtx, allowedUpdate)
	if err != nil {
		t.Fatalf("UpdateLLMConfig() allowed error = %v", err)
	}

	updated, err := repo.GetByID(cfg.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if updated.Name != "Allowed" || updated.Model != "gpt-test-new" {
		t.Fatalf("updated config mismatch: name=%q model=%q", updated.Name, updated.Model)
	}
	if updated.APIKey == "new-secret" || !strings.HasPrefix(updated.APIKey, llmAPIKeyCipherPrefix) {
		t.Fatalf("updated APIKey was not sealed: %q", updated.APIKey)
	}
	opened, err = svc.OpenAPIKey(updated.APIKey)
	if err != nil {
		t.Fatalf("OpenAPIKey(updated) error = %v", err)
	}
	if opened != "new-secret" {
		t.Fatalf("OpenAPIKey(updated) = %q, want new-secret", opened)
	}
	if updated.Admin != "alice,bob" {
		t.Fatalf("Admin after update = %q, want existing admin preserved", updated.Admin)
	}

	blankKeyUpdate := &model.LLMConfig{
		Name:      "Blank Key Preserves",
		Provider:  model.LLMProviderOpenAI,
		Model:     "gpt-test-newer",
		Timeout:   90,
		MaxTokens: 4096,
	}
	blankKeyUpdate.ID = cfg.ID
	err = svc.UpdateLLMConfig(aliceCtx, blankKeyUpdate)
	if err != nil {
		t.Fatalf("UpdateLLMConfig(blank key) error = %v", err)
	}
	preserved, err := repo.GetByID(cfg.ID)
	if err != nil {
		t.Fatalf("GetByID(blank key update) error = %v", err)
	}
	opened, err = svc.OpenAPIKey(preserved.APIKey)
	if err != nil {
		t.Fatalf("OpenAPIKey(preserved) error = %v", err)
	}
	if opened != "new-secret" {
		t.Fatalf("blank APIKey update should preserve existing secret, got %q", opened)
	}
}

func TestLLMConfigDeleteAndSetDefaultRequireAdmin(t *testing.T) {
	svc, repo := newLLMSeedTestService(t)
	aliceCtx := contextx.WithRequestUser(context.Background(), "alice")
	bobCtx := contextx.WithRequestUser(context.Background(), "bob")

	cfg := &model.LLMConfig{
		Name:      "Primary",
		Provider:  model.LLMProviderOpenAI,
		Model:     "gpt-test",
		Admin:     "alice",
		Timeout:   30,
		MaxTokens: 1024,
	}
	if err := svc.CreateLLMConfig(aliceCtx, cfg); err != nil {
		t.Fatalf("CreateLLMConfig() error = %v", err)
	}

	if err := svc.SetDefaultLLMConfig(bobCtx, cfg.ID); err == nil || !strings.Contains(err.Error(), "无权限设置默认") {
		t.Fatalf("SetDefaultLLMConfig() error = %v, want permission error", err)
	}
	if err := svc.SetDefaultLLMConfig(aliceCtx, cfg.ID); err != nil {
		t.Fatalf("SetDefaultLLMConfig() admin error = %v", err)
	}
	defaultCfg, err := repo.GetDefault()
	if err != nil {
		t.Fatalf("GetDefault() error = %v", err)
	}
	if defaultCfg.ID != cfg.ID {
		t.Fatalf("default ID = %d, want %d", defaultCfg.ID, cfg.ID)
	}

	if err := svc.DeleteLLMConfig(bobCtx, cfg.ID); err == nil || !strings.Contains(err.Error(), "无权限删除") {
		t.Fatalf("DeleteLLMConfig() error = %v, want permission error", err)
	}
	if err := svc.DeleteLLMConfig(aliceCtx, cfg.ID); err != nil {
		t.Fatalf("DeleteLLMConfig() admin error = %v", err)
	}
	if _, err := repo.GetByID(cfg.ID); err != gorm.ErrRecordNotFound {
		t.Fatalf("GetByID() after delete error = %v, want ErrRecordNotFound", err)
	}
}

func TestLLMConfigInfersResponsesProtocolFromEndpoint(t *testing.T) {
	svc, repo := newLLMSeedTestService(t)
	aliceCtx := contextx.WithRequestUser(context.Background(), "alice")

	cfg := &model.LLMConfig{
		Name:         "Responses",
		Provider:     model.LLMProviderOpenAI,
		Protocol:     model.LLMProtocolOpenAIChatCompletions,
		Model:        "gpt-test",
		APIBase:      "https://devcloud.chat/api/v1",
		EndpointPath: "/responses",
		Timeout:      30,
		MaxTokens:    1024,
	}
	if err := svc.CreateLLMConfig(aliceCtx, cfg); err != nil {
		t.Fatalf("CreateLLMConfig() error = %v", err)
	}
	stored, err := repo.GetByID(cfg.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if stored.Protocol != model.LLMProtocolOpenAIResponses {
		t.Fatalf("protocol = %q, want openai_responses", stored.Protocol)
	}
}
