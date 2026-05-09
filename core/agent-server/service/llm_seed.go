package service

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"gorm.io/gorm"
)

const defaultLLMSeedAdmin = "system"

// InitLLMSeeds 幂等初始化部署配置中的 LLM。
//
// 只创建或更新带 code 的 seed 配置；不会删除数据库中已存在但配置里移除的 LLM。
func (s *LLMService) InitLLMSeeds(ctx context.Context, seeds config.AgentServerLLMSeedsConfig) error {
	if len(seeds.Configs) == 0 {
		return nil
	}

	defaultCode := strings.TrimSpace(seeds.Default)
	var explicitDefaultCount int
	items := make([]normalizedLLMSeedItem, 0, len(seeds.Configs))
	seenCodes := map[string]struct{}{}

	for _, seed := range seeds.Configs {
		normalized, apiKeySpecified, err := normalizeLLMSeed(seed)
		if err != nil {
			return err
		}
		if _, exists := seenCodes[normalized.Code]; exists {
			return fmt.Errorf("llms.configs code %q 重复", normalized.Code)
		}
		seenCodes[normalized.Code] = struct{}{}
		if normalized.IsDefault {
			explicitDefaultCount++
			if defaultCode == "" {
				defaultCode = normalized.Code
			}
		}
		items = append(items, normalizedLLMSeedItem{seed: normalized, apiKeySpecified: apiKeySpecified})
	}

	if explicitDefaultCount > 1 && strings.TrimSpace(seeds.Default) == "" {
		return fmt.Errorf("llms.configs 中只能有一个 is_default=true")
	}
	if defaultCode != "" {
		found := false
		for _, item := range items {
			if item.seed.Code == defaultCode {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("llms.default=%q 未在 llms.configs 中找到", defaultCode)
		}
	}

	var defaultID int64
	for _, item := range items {
		normalized := item.seed
		existing, err := s.repo.GetByCode(normalized.Code)
		if err != nil && err != gorm.ErrRecordNotFound {
			return fmt.Errorf("查询 LLM seed %q 失败: %w", normalized.Code, err)
		}

		var id int64
		if err == gorm.ErrRecordNotFound {
			cfg := normalized.toModel()
			if err := s.repo.Create(cfg); err != nil {
				return fmt.Errorf("创建 LLM seed %q 失败: %w", normalized.Code, err)
			}
			id = cfg.ID
			logger.Infof(ctx, "[LLMSeed] 已创建 LLM 配置: code=%s provider=%s model=%s", normalized.Code, normalized.Provider, normalized.Model)
		} else {
			applyLLMSeedToModel(existing, normalized, item.apiKeySpecified)
			if err := s.repo.Update(existing); err != nil {
				return fmt.Errorf("更新 LLM seed %q 失败: %w", normalized.Code, err)
			}
			id = existing.ID
			logger.Infof(ctx, "[LLMSeed] 已更新 LLM 配置: code=%s provider=%s model=%s", normalized.Code, normalized.Provider, normalized.Model)
		}

		if normalized.Code == defaultCode {
			defaultID = id
		}
	}

	if defaultCode != "" {
		if defaultID == 0 {
			return fmt.Errorf("llms.default=%q 未在 llms.configs 中找到", defaultCode)
		}
		if err := s.repo.SetDefault(defaultID); err != nil {
			return fmt.Errorf("设置默认 LLM seed %q 失败: %w", defaultCode, err)
		}
		logger.Infof(ctx, "[LLMSeed] 默认 LLM 配置: code=%s id=%d", defaultCode, defaultID)
	}

	return nil
}

type normalizedLLMSeedItem struct {
	seed            normalizedLLMSeed
	apiKeySpecified bool
}

type normalizedLLMSeed struct {
	Code        string
	Name        string
	Provider    string
	Model       string
	APIKey      string
	APIBase     string
	Timeout     int
	MaxTokens   int
	ExtraConfig *string
	UseThinking bool
	IsDefault   bool
	Visibility  int
	Admin       string
}

func normalizeLLMSeed(seed config.AgentServerLLMSeedConfig) (normalizedLLMSeed, bool, error) {
	code := strings.TrimSpace(seed.Code)
	name := strings.TrimSpace(seed.Name)
	provider := strings.TrimSpace(seed.Provider)
	modelName := strings.TrimSpace(seed.Model)
	if code == "" {
		return normalizedLLMSeed{}, false, fmt.Errorf("llms.configs[].code 不能为空")
	}
	if name == "" {
		return normalizedLLMSeed{}, false, fmt.Errorf("llms.configs[%s].name 不能为空", code)
	}
	if provider == "" {
		return normalizedLLMSeed{}, false, fmt.Errorf("llms.configs[%s].provider 不能为空", code)
	}
	if modelName == "" {
		return normalizedLLMSeed{}, false, fmt.Errorf("llms.configs[%s].model 不能为空", code)
	}

	apiKey, apiKeySpecified := resolveLLMSeedAPIKey(seed)
	extraConfig, err := normalizeExtraConfig(strings.TrimSpace(seed.ExtraConfig))
	if err != nil {
		return normalizedLLMSeed{}, false, fmt.Errorf("llms.configs[%s].extra_config 无效: %w", code, err)
	}

	timeout := seed.Timeout
	if timeout <= 0 {
		timeout = defaultLLMTimeout
	}
	maxTokens := seed.MaxTokens
	if maxTokens <= 0 {
		maxTokens = defaultLLMMaxTokens
	}
	admin := strings.TrimSpace(seed.Admin)
	if admin == "" {
		admin = defaultLLMSeedAdmin
	}

	return normalizedLLMSeed{
		Code:        code,
		Name:        name,
		Provider:    provider,
		Model:       modelName,
		APIKey:      apiKey,
		APIBase:     strings.TrimSpace(seed.APIBase),
		Timeout:     timeout,
		MaxTokens:   maxTokens,
		ExtraConfig: extraConfig,
		UseThinking: seed.UseThinking,
		IsDefault:   seed.IsDefault,
		Visibility:  seed.Visibility,
		Admin:       admin,
	}, apiKeySpecified, nil
}

func resolveLLMSeedAPIKey(seed config.AgentServerLLMSeedConfig) (string, bool) {
	apiKeyEnv := strings.TrimSpace(seed.APIKeyEnv)
	if apiKeyEnv != "" {
		if v := strings.TrimSpace(os.Getenv(apiKeyEnv)); v != "" {
			return v, true
		}
		return strings.TrimSpace(seed.APIKey), true
	}
	apiKey := strings.TrimSpace(seed.APIKey)
	return apiKey, apiKey != ""
}

func (seed normalizedLLMSeed) toModel() *model.LLMConfig {
	cfg := &model.LLMConfig{
		Code:        seed.Code,
		Name:        seed.Name,
		Provider:    seed.Provider,
		Model:       seed.Model,
		APIKey:      seed.APIKey,
		APIBase:     seed.APIBase,
		Timeout:     seed.Timeout,
		MaxTokens:   seed.MaxTokens,
		ExtraConfig: seed.ExtraConfig,
		UseThinking: seed.UseThinking,
		Visibility:  seed.Visibility,
		Admin:       seed.Admin,
	}
	cfg.CreatedBy = defaultLLMSeedAdmin
	cfg.UpdatedBy = defaultLLMSeedAdmin
	return cfg
}

func applyLLMSeedToModel(cfg *model.LLMConfig, seed normalizedLLMSeed, apiKeySpecified bool) {
	cfg.Code = seed.Code
	cfg.Name = seed.Name
	cfg.Provider = seed.Provider
	cfg.Model = seed.Model
	if apiKeySpecified && seed.APIKey != "" {
		cfg.APIKey = seed.APIKey
	}
	cfg.APIBase = seed.APIBase
	cfg.Timeout = seed.Timeout
	cfg.MaxTokens = seed.MaxTokens
	cfg.ExtraConfig = seed.ExtraConfig
	cfg.UseThinking = seed.UseThinking
	cfg.Visibility = seed.Visibility
	cfg.Admin = seed.Admin
	cfg.UpdatedBy = defaultLLMSeedAdmin
}
