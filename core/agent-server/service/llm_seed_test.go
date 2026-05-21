package service

import (
	"context"
	"testing"

	"github.com/kageos/kageos/core/agent-server/model"
	"github.com/kageos/kageos/core/agent-server/repository"
	aosconfig "github.com/kageos/kageos/pkg/config"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newLLMSeedTestService(t *testing.T) (*LLMService, *repository.LLMRepository) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.LLMConfig{}); err != nil {
		t.Fatalf("migrate llm configs: %v", err)
	}
	repo := repository.NewLLMRepository(db)
	return NewLLMService(repo), repo
}

func TestInitLLMSeedsCreatesDefaultAndAllowsEmptyAPIBase(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "seed-secret")
	svc, repo := newLLMSeedTestService(t)

	err := svc.InitLLMSeeds(context.Background(), aosconfig.AgentServerLLMSeedsConfig{
		Default: "main",
		Configs: []aosconfig.AgentServerLLMSeedConfig{
			{
				Code:      "main",
				Name:      "默认模型",
				Model:     "gpt-4o-mini",
				APIKeyEnv: "OPENAI_API_KEY",
			},
		},
	})
	if err != nil {
		t.Fatalf("init llm seeds: %v", err)
	}

	cfg, err := repo.GetByCode("main")
	if err != nil {
		t.Fatalf("get seed by code: %v", err)
	}
	if cfg.APIBase != "" {
		t.Fatalf("api_base should stay empty to use SDK defaults, got %q", cfg.APIBase)
	}
	if cfg.APIKey != "seed-secret" {
		t.Fatalf("api key = %q, want env value", cfg.APIKey)
	}
	if cfg.Timeout != defaultLLMTimeout || cfg.MaxTokens != defaultLLMMaxTokens {
		t.Fatalf("defaults timeout/max_tokens = %d/%d", cfg.Timeout, cfg.MaxTokens)
	}
	if cfg.Admin != defaultLLMSeedAdmin {
		t.Fatalf("admin = %q, want %q", cfg.Admin, defaultLLMSeedAdmin)
	}
	def, err := repo.GetDefault()
	if err != nil {
		t.Fatalf("get default: %v", err)
	}
	if def.Code != "main" {
		t.Fatalf("default code = %q, want main", def.Code)
	}
}

func TestInitLLMSeedsDoesNotWipeExistingKeyWhenEnvMissing(t *testing.T) {
	svc, repo := newLLMSeedTestService(t)
	if err := repo.Create(&model.LLMConfig{
		Code:      "main",
		Name:      "旧模型",
		Provider:  "openai",
		Model:     "gpt-4o-mini",
		APIKey:    "existing-secret",
		APIBase:   "https://old.example/v1",
		Timeout:   60,
		MaxTokens: 1024,
		Admin:     "system",
	}); err != nil {
		t.Fatalf("create existing llm: %v", err)
	}

	err := svc.InitLLMSeeds(context.Background(), aosconfig.AgentServerLLMSeedsConfig{
		Configs: []aosconfig.AgentServerLLMSeedConfig{
			{
				Code:      "main",
				Name:      "新模型",
				Model:     "gpt-4.1",
				APIKeyEnv: "MISSING_LLM_API_KEY",
			},
		},
	})
	if err != nil {
		t.Fatalf("init llm seeds: %v", err)
	}

	cfg, err := repo.GetByCode("main")
	if err != nil {
		t.Fatalf("get seed by code: %v", err)
	}
	if cfg.APIKey != "existing-secret" {
		t.Fatalf("api key should not be wiped, got %q", cfg.APIKey)
	}
	if cfg.Name != "新模型" || cfg.Model != "gpt-4.1" {
		t.Fatalf("seed fields not updated: name=%q model=%q", cfg.Name, cfg.Model)
	}
	if cfg.APIBase != "" {
		t.Fatalf("api_base should be updated to empty, got %q", cfg.APIBase)
	}
}

func TestInitLLMSeedsRejectsMissingDefaultCode(t *testing.T) {
	svc, _ := newLLMSeedTestService(t)

	err := svc.InitLLMSeeds(context.Background(), aosconfig.AgentServerLLMSeedsConfig{
		Default: "missing",
		Configs: []aosconfig.AgentServerLLMSeedConfig{
			{
				Code:  "main",
				Name:  "默认模型",
				Model: "gpt-4o-mini",
			},
		},
	})
	if err == nil {
		t.Fatal("expected missing default code to fail")
	}
}
