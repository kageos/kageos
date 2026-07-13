package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/kageos/kageos/core/agent-server/model"
	"github.com/kageos/kageos/core/agent-server/repository"
	"github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/llms"
	"github.com/kageos/kageos/pkg/secretvault"
	"gorm.io/gorm"
)

func normalizeJSONText(label, value string) (*string, error) {
	value = strings.TrimSpace(value)

	// 如果为空，返回 nil（允许 NULL）
	if value == "" {
		return nil, nil
	}

	// 验证是否为有效的 JSON
	var temp interface{}
	if err := json.Unmarshal([]byte(value), &temp); err != nil {
		return nil, fmt.Errorf("%s 不是有效的 JSON: %w", label, err)
	}

	// 重新序列化以确保格式正确
	normalized, err := json.Marshal(temp)
	if err != nil {
		return nil, fmt.Errorf("序列化 %s 失败: %w", label, err)
	}

	result := string(normalized)
	return &result, nil
}

// normalizeExtraConfig 规范化 extra_config 字段，确保是有效的 JSON 或 NULL
func normalizeExtraConfig(extraConfig string) (*string, error) {
	return normalizeJSONText("extra_config", extraConfig)
}

func normalizeOptionalJSONField(label string, value *string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	return normalizeJSONText(label, *value)
}

func normalizeAdminList(admins string) string {
	seen := map[string]struct{}{}
	out := make([]string, 0)
	for _, admin := range strings.Split(admins, ",") {
		admin = strings.TrimSpace(admin)
		if admin == "" {
			continue
		}
		if _, ok := seen[admin]; ok {
			continue
		}
		seen[admin] = struct{}{}
		out = append(out, admin)
	}
	return strings.Join(out, ",")
}

func normalizeLLMProviderProtocol(provider, protocol string) (string, string, error) {
	provider, protocol = llms.NormalizeProviderProtocol(provider, protocol)
	switch protocol {
	case llms.ProtocolOpenAIChatCompletions, llms.ProtocolOpenAIResponses, llms.ProtocolAnthropicMessages:
	default:
		return "", "", fmt.Errorf("不支持的 LLM 协议: %s", protocol)
	}
	switch provider {
	case llms.ProviderOpenAI, llms.ProviderAnthropic:
	default:
		return "", "", fmt.Errorf("不支持的 LLM provider: %s", provider)
	}
	if protocol == llms.ProtocolAnthropicMessages && provider != llms.ProviderAnthropic {
		return "", "", fmt.Errorf("Anthropic Messages 协议必须使用 anthropic provider")
	}
	if protocol != llms.ProtocolAnthropicMessages && provider == llms.ProviderAnthropic {
		return "", "", fmt.Errorf("anthropic provider 仅支持 anthropic_messages 协议")
	}
	return provider, protocol, nil
}

func (s *LLMService) getManageableLLMConfig(ctx context.Context, id int64, action string) (*model.LLMConfig, error) {
	cfg, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("LLM配置不存在")
		}
		return nil, fmt.Errorf("获取LLM配置失败: %w", err)
	}
	if !cfg.IsAdminUser(contextx.GetRequestUser(ctx)) {
		return nil, fmt.Errorf("无权限%s该LLM配置", action)
	}
	return cfg, nil
}

// LLMService LLM 服务
type LLMService struct {
	repo           *repository.LLMRepository
	apiKeyVault    *secretvault.Vault
	apiKeyVaultErr error
}

const (
	defaultLLMTimeout     = 300
	defaultLLMMaxTokens   = 8196
	llmAPIKeySecretEnv    = "KAGEOS_LLM_API_KEY_SECRET"
	llmAPIKeyVaultPurpose = "kageos-agent-llm-api-key-v1"
	llmAPIKeyCipherPrefix = "kgosecret:llm-api-key:v1:"
)

type LLMServiceOption func(*LLMService)

func WithLLMAPIKeySecret(secret string) LLMServiceOption {
	return func(s *LLMService) {
		vault, err := newLLMAPIKeyVault(secret)
		s.apiKeyVault = vault
		s.apiKeyVaultErr = err
	}
}

// NewLLMService 创建 LLM 服务
func NewLLMService(repo *repository.LLMRepository, opts ...LLMServiceOption) *LLMService {
	s := &LLMService{repo: repo}
	WithLLMAPIKeySecret(defaultLLMAPIKeySecret())(s)
	for _, opt := range opts {
		if opt != nil {
			opt(s)
		}
	}
	return s
}

// GetLLMConfig 获取 LLM 配置
func (s *LLMService) GetLLMConfig(ctx context.Context, id int64) (*model.LLMConfig, error) {
	cfg, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("LLM配置不存在")
		}
		return nil, fmt.Errorf("获取LLM配置失败: %w", err)
	}
	return cfg, nil
}

func (s *LLMService) GetViewableLLMConfig(ctx context.Context, id int64) (*model.LLMConfig, error) {
	cfg, err := s.GetLLMConfig(ctx, id)
	if err != nil {
		return nil, err
	}
	if !canViewLLMConfig(cfg, contextx.GetRequestUser(ctx)) {
		return nil, fmt.Errorf("无权限查看该LLM配置")
	}
	return cfg, nil
}

// GetDefaultLLMConfig 获取默认 LLM 配置
func (s *LLMService) GetDefaultLLMConfig(ctx context.Context) (*model.LLMConfig, error) {
	cfg, err := s.repo.GetDefault(ctx)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("未设置默认LLM配置")
		}
		return nil, fmt.Errorf("获取默认LLM配置失败: %w", err)
	}
	return cfg, nil
}

func (s *LLMService) GetViewableDefaultLLMConfig(ctx context.Context) (*model.LLMConfig, error) {
	cfg, err := s.GetDefaultLLMConfig(ctx)
	if err != nil {
		return nil, err
	}
	if !canViewLLMConfig(cfg, contextx.GetRequestUser(ctx)) {
		return nil, fmt.Errorf("无权限查看默认LLM配置")
	}
	return cfg, nil
}

// ListLLMConfigs 获取 LLM 配置列表
func (s *LLMService) ListLLMConfigs(ctx context.Context, scope string, page, pageSize int) ([]*model.LLMConfig, int64, error) {
	currentUser := contextx.GetRequestUser(ctx)
	offset := (page - 1) * pageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	return s.repo.List(ctx, scope, currentUser, offset, pageSize)
}

// CreateLLMConfig 创建 LLM 配置
func (s *LLMService) CreateLLMConfig(ctx context.Context, cfg *model.LLMConfig) error {
	// 获取用户信息
	user := contextx.GetRequestUser(ctx)
	cfg.CreatedBy = user
	cfg.UpdatedBy = user

	// 验证必填字段
	if cfg.Name == "" {
		return fmt.Errorf("配置名称不能为空")
	}
	provider, protocol, err := normalizeLLMProviderProtocol(cfg.Provider, cfg.Protocol)
	if err != nil {
		return err
	}
	provider, protocol = llms.InferProviderProtocol(provider, protocol, cfg.APIBase, cfg.EndpointPath)
	cfg.Provider = provider
	cfg.Protocol = protocol
	if cfg.Model == "" {
		return fmt.Errorf("模型名称不能为空")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultLLMTimeout
	}
	if cfg.MaxTokens <= 0 {
		cfg.MaxTokens = defaultLLMMaxTokens
	}

	// 规范化 extra_config 字段
	normalizedExtraConfig, err := normalizeExtraConfig(func() string {
		if cfg.ExtraConfig != nil {
			return *cfg.ExtraConfig
		}
		return ""
	}())
	if err != nil {
		return err
	}
	cfg.ExtraConfig = normalizedExtraConfig
	cfg.Headers, err = normalizeOptionalJSONField("headers", cfg.Headers)
	if err != nil {
		return err
	}
	cfg.Capabilities, err = normalizeOptionalJSONField("capabilities", cfg.Capabilities)
	if err != nil {
		return err
	}

	// 设置默认管理员（如果为空，设置为创建用户）
	if cfg.Admin == "" {
		cfg.Admin = user
	}
	cfg.Admin = normalizeAdminList(cfg.Admin)

	if err := s.sealConfigAPIKey(cfg); err != nil {
		return err
	}

	// 先创建配置
	if err := s.repo.Create(ctx, cfg); err != nil {
		return err
	}

	// 如果设置为默认，设置默认配置
	if cfg.IsDefault {
		if err := s.repo.SetDefault(ctx, cfg.ID); err != nil {
			return fmt.Errorf("设置默认配置失败: %w", err)
		}
	}

	return nil
}

// UpdateLLMConfig 更新 LLM 配置
func (s *LLMService) UpdateLLMConfig(ctx context.Context, cfg *model.LLMConfig) error {
	// 获取用户信息
	user := contextx.GetRequestUser(ctx)
	cfg.UpdatedBy = user

	// 检查权限：只有管理员可以修改资源
	existing, err := s.getManageableLLMConfig(ctx, cfg.ID, "修改")
	if err != nil {
		return err
	}

	// 验证必填字段
	if cfg.Name == "" {
		return fmt.Errorf("配置名称不能为空")
	}
	provider, protocol, err := normalizeLLMProviderProtocol(cfg.Provider, cfg.Protocol)
	if err != nil {
		return err
	}
	provider, protocol = llms.InferProviderProtocol(provider, protocol, cfg.APIBase, cfg.EndpointPath)
	cfg.Provider = provider
	cfg.Protocol = protocol
	if cfg.Model == "" {
		return fmt.Errorf("模型名称不能为空")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultLLMTimeout
	}
	if cfg.MaxTokens <= 0 {
		cfg.MaxTokens = defaultLLMMaxTokens
	}

	// 规范化 extra_config 字段
	normalizedExtraConfig, err := normalizeExtraConfig(func() string {
		if cfg.ExtraConfig != nil {
			return *cfg.ExtraConfig
		}
		return ""
	}())
	if err != nil {
		return err
	}
	cfg.ExtraConfig = normalizedExtraConfig
	cfg.Headers, err = normalizeOptionalJSONField("headers", cfg.Headers)
	if err != nil {
		return err
	}
	cfg.Capabilities, err = normalizeOptionalJSONField("capabilities", cfg.Capabilities)
	if err != nil {
		return err
	}

	if cfg.Admin == "" {
		if existing.Admin != "" {
			cfg.Admin = existing.Admin
		} else {
			cfg.Admin = user
		}
	}
	cfg.Admin = normalizeAdminList(cfg.Admin)
	if strings.TrimSpace(cfg.APIKey) == "" {
		cfg.APIKey = existing.APIKey
	}
	if err := s.sealConfigAPIKey(cfg); err != nil {
		return err
	}

	// 如果设置为默认，先取消其他默认配置
	if cfg.IsDefault {
		if err := s.repo.SetDefault(ctx, cfg.ID); err != nil {
			return fmt.Errorf("设置默认配置失败: %w", err)
		}
	}

	return s.repo.Update(ctx, cfg)
}

// DeleteLLMConfig 删除 LLM 配置
func (s *LLMService) DeleteLLMConfig(ctx context.Context, id int64) error {
	if _, err := s.getManageableLLMConfig(ctx, id, "删除"); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

// SetDefaultLLMConfig 设置默认 LLM 配置
func (s *LLMService) SetDefaultLLMConfig(ctx context.Context, id int64) error {
	if _, err := s.getManageableLLMConfig(ctx, id, "设置默认"); err != nil {
		return err
	}
	return s.repo.SetDefault(ctx, id)
}

func canViewLLMConfig(cfg *model.LLMConfig, username string) bool {
	if cfg == nil {
		return false
	}
	return cfg.Visibility == 0 || cfg.IsAdminUser(username)
}

func defaultLLMAPIKeySecret() string {
	if value := strings.TrimSpace(os.Getenv(llmAPIKeySecretEnv)); value != "" {
		return value
	}
	if value := strings.TrimSpace(config.GetGlobalSharedConfig().JWT.Secret); value != "" {
		return value
	}
	return ""
}

func newLLMAPIKeyVault(secret string) (*secretvault.Vault, error) {
	return secretvault.New(
		secret,
		llmAPIKeyVaultPurpose,
		secretvault.WithPrefix(llmAPIKeyCipherPrefix),
	)
}

func (s *LLMService) sealConfigAPIKey(cfg *model.LLMConfig) error {
	if cfg == nil || strings.TrimSpace(cfg.APIKey) == "" {
		return nil
	}
	sealed, err := sealLLMAPIKey(s.apiKeyVault, s.apiKeyVaultErr, cfg.APIKey)
	if err != nil {
		return err
	}
	cfg.APIKey = sealed
	return nil
}

func (s *LLMService) OpenAPIKey(value string) (string, error) {
	return openLLMAPIKey(s.apiKeyVault, s.apiKeyVaultErr, value)
}

func sealLLMAPIKey(vault *secretvault.Vault, vaultErr error, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if vaultErr != nil {
		return "", fmt.Errorf("初始化 LLM API Key 加密失败: %w", vaultErr)
	}
	if vault == nil {
		return "", fmt.Errorf("LLM API Key 加密器未初始化")
	}
	return vault.Seal(value)
}

func openLLMAPIKey(vault *secretvault.Vault, vaultErr error, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if vaultErr != nil {
		return "", fmt.Errorf("初始化 LLM API Key 解密失败: %w", vaultErr)
	}
	if vault == nil {
		return "", fmt.Errorf("LLM API Key 解密器未初始化")
	}
	return vault.Open(value)
}

func isSealedLLMAPIKey(vault *secretvault.Vault, value string) bool {
	return vault != nil && vault.IsSealed(strings.TrimSpace(value))
}
