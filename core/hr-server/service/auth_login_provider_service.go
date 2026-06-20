package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/kageos/kageos/core/hr-server/model"
	"github.com/kageos/kageos/core/hr-server/repository"
	"github.com/kageos/kageos/dto"
)

const (
	ProviderGoogleOAuth     = "google_oauth"
	ProviderGitHubOAuth     = "github_oauth"
	ProviderWechatMPQR      = "wechat_mp_qr"
	ProviderWechatOpenLogin = "wechat_open_login"

	ProviderActionRedirect = "redirect"
	ProviderActionQRCode   = "qrcode"

	ProviderStatusUnconfigured = "unconfigured"
	ProviderStatusDisabled     = "disabled"
	ProviderStatusEnabled      = "enabled"
)

type AuthLoginProviderService struct {
	repo *repository.AuthLoginProviderRepository
}

type authProviderFieldDef struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Secret      bool   `json:"secret"`
	Help        string `json:"help,omitempty"`
	Placeholder string `json:"placeholder,omitempty"`
}

type authProviderSeed struct {
	Code         string
	Name         string
	Description  string
	Action       string
	Fields       []authProviderFieldDef
	CallbackPath string
	DocsURL      string
	SortOrder    int
}

type AuthLoginProviderRuntimeConfig struct {
	Code   string
	Action string
	Values map[string]string
}

func NewAuthLoginProviderService(repo *repository.AuthLoginProviderRepository) *AuthLoginProviderService {
	return &AuthLoginProviderService{repo: repo}
}

func (s *AuthLoginProviderService) SeedDefaults(ctx context.Context) error {
	for _, seed := range defaultAuthProviderSeeds() {
		schema, err := json.Marshal(seed.Fields)
		if err != nil {
			return err
		}
		if err := s.repo.UpsertSeed(&model.AuthLoginProvider{
			Code:             seed.Code,
			Name:             seed.Name,
			Description:      seed.Description,
			Action:           seed.Action,
			Enabled:          false,
			Configured:       false,
			Status:           ProviderStatusUnconfigured,
			ConfigSchemaJSON: string(schema),
			ConfigValuesJSON: "{}",
			CallbackPath:     seed.CallbackPath,
			DocsURL:          seed.DocsURL,
			SortOrder:        seed.SortOrder,
			UpdatedBy:        "system",
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *AuthLoginProviderService) ListProviders() ([]*dto.AuthLoginProviderResp, error) {
	providers, err := s.repo.List()
	if err != nil {
		return nil, err
	}
	out := make([]*dto.AuthLoginProviderResp, 0, len(providers))
	for _, provider := range providers {
		resp, err := s.toProviderResp(provider)
		if err != nil {
			return nil, err
		}
		out = append(out, resp)
	}
	return out, nil
}

func (s *AuthLoginProviderService) ListLoginMethods() ([]dto.LoginMethodResp, error) {
	providers, err := s.ListProviders()
	if err != nil {
		return nil, err
	}
	methods := make([]dto.LoginMethodResp, 0, len(providers))
	for _, provider := range providers {
		if provider.Enabled && provider.Configured && provider.Status == ProviderStatusEnabled {
			methods = append(methods, dto.LoginMethodResp{
				Provider:      provider.Code,
				Label:         provider.Name,
				Action:        provider.Action,
				Description:   provider.Description,
				AuthorizePath: providerAuthorizePath(provider.Code),
			})
		}
	}
	return methods, nil
}

func (s *AuthLoginProviderService) GetEnabledRuntimeConfig(code string) (*AuthLoginProviderRuntimeConfig, error) {
	provider, err := s.repo.GetByCode(normalizeProviderCode(code))
	if err != nil {
		return nil, err
	}
	if provider == nil {
		return nil, fmt.Errorf("login provider %q not found", code)
	}
	fields, err := parseProviderFields(provider.ConfigSchemaJSON)
	if err != nil {
		return nil, err
	}
	values := parseProviderValues(provider.ConfigValuesJSON)
	if !provider.Enabled || !providerConfigured(fields, values) {
		return nil, fmt.Errorf("login provider %q is not enabled", code)
	}
	return &AuthLoginProviderRuntimeConfig{
		Code:   provider.Code,
		Action: provider.Action,
		Values: values,
	}, nil
}

func (s *AuthLoginProviderService) UpdateConfig(code string, config map[string]string, updatedBy string) (*dto.AuthLoginProviderResp, error) {
	provider, err := s.repo.GetByCode(normalizeProviderCode(code))
	if err != nil {
		return nil, err
	}
	if provider == nil {
		return nil, fmt.Errorf("login provider %q not found", code)
	}

	fields, err := parseProviderFields(provider.ConfigSchemaJSON)
	if err != nil {
		return nil, err
	}
	values := parseProviderValues(provider.ConfigValuesJSON)
	for _, field := range fields {
		raw, exists := config[field.Key]
		if !exists {
			continue
		}
		value := strings.TrimSpace(raw)
		if field.Secret && value == "" {
			continue
		}
		if value == "" {
			delete(values, field.Key)
			continue
		}
		values[field.Key] = value
	}

	configured := providerConfigured(fields, values)
	provider.Configured = configured
	if !configured {
		provider.Enabled = false
		provider.Status = ProviderStatusUnconfigured
	} else if provider.Enabled {
		provider.Status = ProviderStatusEnabled
	} else {
		provider.Status = ProviderStatusDisabled
	}
	provider.UpdatedBy = strings.TrimSpace(updatedBy)
	valuesJSON, err := json.Marshal(values)
	if err != nil {
		return nil, err
	}
	provider.ConfigValuesJSON = string(valuesJSON)
	if err := s.repo.UpdateConfig(provider); err != nil {
		return nil, err
	}
	updated, err := s.repo.GetByCode(provider.Code)
	if err != nil {
		return nil, err
	}
	return s.toProviderResp(updated)
}

func (s *AuthLoginProviderService) SetEnabled(code string, enabled bool, updatedBy string) (*dto.AuthLoginProviderResp, error) {
	provider, err := s.repo.GetByCode(normalizeProviderCode(code))
	if err != nil {
		return nil, err
	}
	if provider == nil {
		return nil, fmt.Errorf("login provider %q not found", code)
	}
	fields, err := parseProviderFields(provider.ConfigSchemaJSON)
	if err != nil {
		return nil, err
	}
	values := parseProviderValues(provider.ConfigValuesJSON)
	configured := providerConfigured(fields, values)
	if enabled && !configured {
		return nil, fmt.Errorf("login provider %q is not configured", code)
	}
	status := ProviderStatusDisabled
	if !configured {
		status = ProviderStatusUnconfigured
		enabled = false
	} else if enabled {
		status = ProviderStatusEnabled
	}
	if err := s.repo.UpdateEnabled(provider.Code, enabled, status, strings.TrimSpace(updatedBy)); err != nil {
		return nil, err
	}
	updated, err := s.repo.GetByCode(provider.Code)
	if err != nil {
		return nil, err
	}
	return s.toProviderResp(updated)
}

func (s *AuthLoginProviderService) toProviderResp(provider *model.AuthLoginProvider) (*dto.AuthLoginProviderResp, error) {
	fields, err := parseProviderFields(provider.ConfigSchemaJSON)
	if err != nil {
		return nil, err
	}
	values := parseProviderValues(provider.ConfigValuesJSON)
	configured := providerConfigured(fields, values)
	status := provider.Status
	if !configured {
		status = ProviderStatusUnconfigured
	} else if provider.Enabled {
		status = ProviderStatusEnabled
	} else if status == "" || status == ProviderStatusUnconfigured {
		status = ProviderStatusDisabled
	}

	fieldResp := make([]dto.AuthLoginProviderField, 0, len(fields))
	for _, field := range fields {
		value := strings.TrimSpace(values[field.Key])
		item := dto.AuthLoginProviderField{
			Key:         field.Key,
			Label:       field.Label,
			Type:        field.Type,
			Required:    field.Required,
			Secret:      field.Secret,
			Help:        field.Help,
			Placeholder: field.Placeholder,
			ValueSet:    value != "",
		}
		if !field.Secret {
			item.Value = value
		}
		fieldResp = append(fieldResp, item)
	}

	return &dto.AuthLoginProviderResp{
		Code:         provider.Code,
		Name:         provider.Name,
		Description:  provider.Description,
		Action:       provider.Action,
		Enabled:      provider.Enabled && configured,
		Configured:   configured,
		Status:       status,
		CallbackPath: provider.CallbackPath,
		DocsURL:      provider.DocsURL,
		Fields:       fieldResp,
		UpdatedBy:    provider.UpdatedBy,
		UpdatedAt:    time.Time(provider.UpdatedAt).Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

func parseProviderFields(raw string) ([]authProviderFieldDef, error) {
	var fields []authProviderFieldDef
	if strings.TrimSpace(raw) == "" {
		return fields, nil
	}
	if err := json.Unmarshal([]byte(raw), &fields); err != nil {
		return nil, err
	}
	return fields, nil
}

func parseProviderValues(raw string) map[string]string {
	values := map[string]string{}
	if strings.TrimSpace(raw) == "" {
		return values
	}
	_ = json.Unmarshal([]byte(raw), &values)
	return values
}

func providerConfigured(fields []authProviderFieldDef, values map[string]string) bool {
	for _, field := range fields {
		if field.Required && strings.TrimSpace(values[field.Key]) == "" {
			return false
		}
	}
	return true
}

func normalizeProviderCode(code string) string {
	return strings.ToLower(strings.TrimSpace(code))
}

func providerAuthorizePath(code string) string {
	switch normalizeProviderCode(code) {
	case ProviderGoogleOAuth:
		return "/hr/api/v1/auth/google/authorize"
	case ProviderGitHubOAuth:
		return "/hr/api/v1/auth/github/authorize"
	default:
		return ""
	}
}

func defaultAuthProviderSeeds() []authProviderSeed {
	seeds := []authProviderSeed{
		{
			Code:         ProviderGoogleOAuth,
			Name:         "Google 登录",
			Description:  "使用 Google OAuth 账号登录。",
			Action:       ProviderActionRedirect,
			CallbackPath: "/hr/api/v1/auth/google/callback",
			DocsURL:      "https://developers.google.com/identity/protocols/oauth2",
			SortOrder:    10,
			Fields: []authProviderFieldDef{
				{Key: "client_id", Label: "Client ID", Type: "text", Required: true},
				{Key: "client_secret", Label: "Client Secret", Type: "password", Required: true, Secret: true},
				{Key: "redirect_url", Label: "Redirect URL", Type: "url", Required: true, Help: "需要和 Google Cloud Console 中配置的回调地址一致。"},
				{Key: "default_company_code", Label: "默认企业代码", Type: "text", Required: false, Placeholder: model.DefaultCompanyCode, Help: "授权注册创建用户时归属的企业代码；留空使用 default。"},
			},
		},
		{
			Code:         ProviderGitHubOAuth,
			Name:         "GitHub 登录",
			Description:  "使用 GitHub OAuth 账号登录。",
			Action:       ProviderActionRedirect,
			CallbackPath: "/hr/api/v1/auth/github/callback",
			DocsURL:      "https://docs.github.com/apps/oauth-apps",
			SortOrder:    20,
			Fields: []authProviderFieldDef{
				{Key: "client_id", Label: "Client ID", Type: "text", Required: true},
				{Key: "client_secret", Label: "Client Secret", Type: "password", Required: true, Secret: true},
				{Key: "redirect_url", Label: "Redirect URL", Type: "url", Required: true, Help: "需要和 GitHub OAuth App 中配置的 Authorization callback URL 一致。"},
				{Key: "scopes", Label: "Scopes", Type: "text", Required: false, Placeholder: "read:user user:email"},
				{Key: "default_company_code", Label: "默认企业代码", Type: "text", Required: false, Placeholder: model.DefaultCompanyCode, Help: "授权注册创建用户时归属的企业代码；留空使用 default。"},
			},
		},
		{
			Code:         ProviderWechatMPQR,
			Name:         "微信公众号扫码关注登录",
			Description:  "使用认证服务号带参数二维码，支持扫码关注后登录或绑定。",
			Action:       ProviderActionQRCode,
			CallbackPath: "/hr/api/v1/auth/wechat_mp/events",
			DocsURL:      "https://developers.weixin.qq.com/doc/service/api/qrcode/qrcodes/api_createqrcode.html",
			SortOrder:    30,
			Fields: []authProviderFieldDef{
				{Key: "appid", Label: "AppID", Type: "text", Required: true},
				{Key: "appsecret", Label: "AppSecret", Type: "password", Required: true, Secret: true},
				{Key: "token", Label: "Token", Type: "password", Required: true, Secret: true},
				{Key: "aes_key", Label: "EncodingAESKey", Type: "password", Required: false, Secret: true},
				{Key: "callback_url", Label: "服务器回调 URL", Type: "url", Required: true, Help: "需要填入微信公众平台「服务器配置」中。"},
			},
		},
		{
			Code:         ProviderWechatOpenLogin,
			Name:         "微信授权登录",
			Description:  "使用微信开放平台网站应用授权登录。",
			Action:       ProviderActionRedirect,
			CallbackPath: "/hr/api/v1/auth/wechat_open/callback",
			DocsURL:      "https://developers.weixin.qq.com/doc/oplatform/Website_App/WeChat_Login/Wechat_Login.html",
			SortOrder:    40,
			Fields: []authProviderFieldDef{
				{Key: "appid", Label: "AppID", Type: "text", Required: true},
				{Key: "appsecret", Label: "AppSecret", Type: "password", Required: true, Secret: true},
				{Key: "redirect_url", Label: "Redirect URL", Type: "url", Required: true, Help: "需要和微信开放平台网站应用授权回调域一致。"},
			},
		},
	}
	sort.Slice(seeds, func(i, j int) bool {
		return seeds[i].SortOrder < seeds[j].SortOrder
	})
	return seeds
}
