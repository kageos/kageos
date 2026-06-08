package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/kageos/kageos/core/connector-server/model"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/contextx"
	"gorm.io/gorm"
)

func (s *ConnectorService) ListOAuthProviders(ctx context.Context) ([]dto.ConnectorOAuthProviderInfo, error) {
	if err := s.ensureOAuthReady(); err != nil {
		return nil, err
	}
	rows, err := s.repo.ListOAuthProviderSettings(ctx)
	if err != nil {
		return nil, err
	}
	providers := s.oauth.List()
	managed := make(map[string]*model.ConnectorOAuthProviderSetting, len(rows))
	for _, row := range rows {
		managed[normalizeProvider(row.Code)] = row
		provider, err := s.providerSettingToConfig(row)
		if err != nil {
			return nil, err
		}
		providers[provider.Code] = mergeOAuthProvider(providers[provider.Code], provider)
	}
	items := make([]dto.ConnectorOAuthProviderInfo, 0, len(providers))
	for code, provider := range providers {
		info := providerConfigToInfo(provider)
		if row := managed[code]; row != nil {
			info = providerConfigToInfo(provider)
			applyProviderSettingMetadata(&info, row)
			info.Managed = true
		}
		items = append(items, info)
	}
	sortProviderInfos(items)
	return items, nil
}

func (s *ConnectorService) GetOAuthProvider(ctx context.Context, code string) (*dto.ConnectorOAuthProviderInfo, error) {
	if err := s.ensureOAuthReady(); err != nil {
		return nil, err
	}
	code = normalizeProvider(code)
	if code == "" {
		return nil, fmt.Errorf("provider code 不能为空")
	}
	row, err := s.repo.GetOAuthProviderSetting(ctx, code)
	if err == nil {
		provider, err := s.providerSettingToConfig(row)
		if err != nil {
			return nil, err
		}
		if base, ok := s.oauth.Lookup(code); ok {
			provider = mergeOAuthProvider(base, provider)
		}
		info := providerConfigToInfo(provider)
		applyProviderSettingMetadata(&info, row)
		return &info, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	provider, ok := s.oauth.Lookup(code)
	if !ok {
		return nil, fmt.Errorf("未配置 OAuth provider: %s", code)
	}
	info := providerConfigToInfo(provider)
	return &info, nil
}

func (s *ConnectorService) SeedOAuthProviderSettings(ctx context.Context) error {
	if err := s.ensureOAuthReady(); err != nil {
		return err
	}
	providers := s.oauth.List()
	codes := make([]string, 0, len(providers))
	for code := range providers {
		codes = append(codes, code)
	}
	sort.Strings(codes)

	for _, code := range codes {
		setting, err := s.providerConfigToSeedSetting(providers[code])
		if err != nil {
			return err
		}
		if err := s.repo.CreateOAuthProviderSettingIfNotExists(ctx, setting); err != nil {
			return err
		}
		if err := s.repo.FillOAuthProviderDisplayDefaults(ctx, code, setting.ProviderAccountURL, setting.LogoURL, setting.BrandColor); err != nil {
			return err
		}
	}
	return nil
}

func (s *ConnectorService) UpsertOAuthProvider(ctx context.Context, req dto.UpsertConnectorOAuthProviderReq) (*dto.ConnectorOAuthProviderInfo, error) {
	if err := s.ensureOAuthReady(); err != nil {
		return nil, err
	}
	user := contextx.GetRequestUser(ctx)
	if err := s.requireProviderAdmin(user); err != nil {
		return nil, err
	}
	code := normalizeProvider(req.Code)
	if code == "" {
		return nil, fmt.Errorf("provider code 不能为空")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = code
	}
	existing, err := s.repo.GetOAuthProviderSetting(ctx, code)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	current := config.ConnectorOAuthProviderConfig{Code: code}
	if base, ok := s.oauth.Lookup(code); ok {
		current = base
	}
	if existing != nil {
		existingProvider, err := s.providerSettingToConfig(existing)
		if err != nil {
			return nil, err
		}
		current = mergeOAuthProvider(current, existingProvider)
	}
	req = mergeOAuthProviderUpsertReq(current, req)
	authType, err := normalizeConnectorAuthType(req.AuthType)
	if err != nil {
		return nil, err
	}
	clientSecretCipher := ""
	if existing != nil {
		clientSecretCipher = existing.ClientSecretCipher
	}
	if strings.TrimSpace(req.ClientSecret) != "" {
		clientSecretCipher, err = s.tokenVault.Seal(req.ClientSecret)
		if err != nil {
			return nil, err
		}
	}
	extraAuth, err := marshalStringMap(current.ExtraAuthParams)
	if err != nil {
		return nil, fmt.Errorf("extra_auth_params 非法: %w", err)
	}
	extraToken, err := marshalStringMap(current.ExtraTokenParams)
	if err != nil {
		return nil, fmt.Errorf("extra_token_params 非法: %w", err)
	}
	enabled := true
	if existing != nil {
		enabled = existing.Enabled
	}
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	setting := &model.ConnectorOAuthProviderSetting{
		Code:               code,
		Name:               name,
		AuthType:           authType,
		ClientID:           strings.TrimSpace(req.ClientID),
		ClientSecretCipher: clientSecretCipher,
		AuthURL:            strings.TrimSpace(req.AuthURL),
		TokenURL:           strings.TrimSpace(req.TokenURL),
		UserInfoURL:        strings.TrimSpace(current.UserInfoURL),
		Scopes:             strings.Join(cleanScopes(req.Scopes), " "),
		UsePKCE:            current.UsePKCE,
		TokenRequestMode:   normalizeProvider(current.TokenRequestMode),
		ClientIDParam:      strings.TrimSpace(current.ClientIDParam),
		ClientSecretParam:  strings.TrimSpace(current.ClientSecretParam),
		GrantTypeParam:     strings.TrimSpace(current.GrantTypeParam),
		CodeParam:          strings.TrimSpace(current.CodeParam),
		RefreshTokenParam:  strings.TrimSpace(current.RefreshTokenParam),
		RedirectURIParam:   strings.TrimSpace(current.RedirectURIParam),
		ExtraAuthParams:    extraAuth,
		ExtraTokenParams:   extraToken,
		ExternalIDField:    strings.TrimSpace(current.ExternalIDField),
		DisplayNameField:   strings.TrimSpace(current.DisplayNameField),
		ProviderAccountURL: strings.TrimSpace(req.ProviderAccountURL),
		LogoURL:            strings.TrimSpace(req.LogoURL),
		BrandColor:         strings.TrimSpace(req.BrandColor),
		Enabled:            enabled,
	}
	setting.CreatedBy = user
	setting.UpdatedBy = user
	if err := s.repo.UpsertOAuthProviderSetting(ctx, setting); err != nil {
		return nil, err
	}
	saved, err := s.repo.GetOAuthProviderSetting(ctx, code)
	if err != nil {
		return nil, err
	}
	info, err := s.providerSettingToMergedInfo(saved)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

func (s *ConnectorService) DeleteOAuthProvider(ctx context.Context, code string) error {
	user := contextx.GetRequestUser(ctx)
	if err := s.requireProviderAdmin(user); err != nil {
		return err
	}
	code = normalizeProvider(code)
	if code == "" {
		return fmt.Errorf("provider code 不能为空")
	}
	rows, err := s.repo.DeleteOAuthProviderSetting(ctx, code)
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("provider 配置不存在")
	}
	return nil
}

func (s *ConnectorService) resolveOAuthProvider(ctx context.Context, code string) (config.ConnectorOAuthProviderConfig, error) {
	code = normalizeProvider(code)
	base, ok := s.oauth.Lookup(code)
	if !ok {
		base = config.ConnectorOAuthProviderConfig{Code: code}
	}
	row, err := s.repo.GetOAuthProviderSetting(ctx, code)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return s.oauth.Get(code)
		}
		return config.ConnectorOAuthProviderConfig{}, err
	}
	if !row.Enabled {
		return config.ConnectorOAuthProviderConfig{}, fmt.Errorf("OAuth provider 已禁用: %s", code)
	}
	override, err := s.providerSettingToConfig(row)
	if err != nil {
		return config.ConnectorOAuthProviderConfig{}, err
	}
	provider := mergeOAuthProvider(base, override)
	return validateOAuthProvider(provider)
}

func (s *ConnectorService) providerSettingToConfig(row *model.ConnectorOAuthProviderSetting) (config.ConnectorOAuthProviderConfig, error) {
	if row == nil {
		return config.ConnectorOAuthProviderConfig{}, fmt.Errorf("provider setting 不能为空")
	}
	clientSecret, err := s.tokenVault.Open(row.ClientSecretCipher)
	if err != nil {
		return config.ConnectorOAuthProviderConfig{}, err
	}
	extraAuth, err := unmarshalStringMap(row.ExtraAuthParams)
	if err != nil {
		return config.ConnectorOAuthProviderConfig{}, err
	}
	extraToken, err := unmarshalStringMap(row.ExtraTokenParams)
	if err != nil {
		return config.ConnectorOAuthProviderConfig{}, err
	}
	return config.ConnectorOAuthProviderConfig{
		Code:               normalizeProvider(row.Code),
		Name:               strings.TrimSpace(row.Name),
		AuthType:           defaultConnectorAuthType(row.AuthType),
		ClientID:           strings.TrimSpace(row.ClientID),
		ClientSecret:       clientSecret,
		AuthURL:            strings.TrimSpace(row.AuthURL),
		TokenURL:           strings.TrimSpace(row.TokenURL),
		UserInfoURL:        strings.TrimSpace(row.UserInfoURL),
		Scopes:             splitScopes(row.Scopes),
		UsePKCE:            row.UsePKCE,
		TokenRequestMode:   strings.TrimSpace(row.TokenRequestMode),
		ClientIDParam:      strings.TrimSpace(row.ClientIDParam),
		ClientSecretParam:  strings.TrimSpace(row.ClientSecretParam),
		GrantTypeParam:     strings.TrimSpace(row.GrantTypeParam),
		CodeParam:          strings.TrimSpace(row.CodeParam),
		RefreshTokenParam:  strings.TrimSpace(row.RefreshTokenParam),
		RedirectURIParam:   strings.TrimSpace(row.RedirectURIParam),
		ExtraAuthParams:    extraAuth,
		ExtraTokenParams:   extraToken,
		ExternalIDField:    strings.TrimSpace(row.ExternalIDField),
		DisplayNameField:   strings.TrimSpace(row.DisplayNameField),
		ProviderAccountURL: strings.TrimSpace(row.ProviderAccountURL),
		LogoURL:            strings.TrimSpace(row.LogoURL),
		BrandColor:         strings.TrimSpace(row.BrandColor),
	}, nil
}

func mergeOAuthProviderUpsertReq(current config.ConnectorOAuthProviderConfig, req dto.UpsertConnectorOAuthProviderReq) dto.UpsertConnectorOAuthProviderReq {
	if strings.TrimSpace(req.Name) == "" {
		req.Name = current.Name
	}
	if strings.TrimSpace(req.AuthType) == "" {
		req.AuthType = current.AuthType
	}
	if strings.TrimSpace(req.ClientID) == "" {
		req.ClientID = current.ClientID
	}
	if strings.TrimSpace(req.AuthURL) == "" {
		req.AuthURL = current.AuthURL
	}
	if strings.TrimSpace(req.TokenURL) == "" {
		req.TokenURL = current.TokenURL
	}
	if req.Scopes == nil {
		req.Scopes = current.Scopes
	}
	if strings.TrimSpace(req.ProviderAccountURL) == "" {
		req.ProviderAccountURL = current.ProviderAccountURL
	}
	if strings.TrimSpace(req.LogoURL) == "" {
		req.LogoURL = current.LogoURL
	}
	if strings.TrimSpace(req.BrandColor) == "" {
		req.BrandColor = current.BrandColor
	}
	return req
}

func (s *ConnectorService) providerConfigToSeedSetting(provider config.ConnectorOAuthProviderConfig) (*model.ConnectorOAuthProviderSetting, error) {
	code := normalizeProvider(provider.Code)
	if code == "" {
		return nil, fmt.Errorf("provider code 不能为空")
	}
	authType, err := normalizeConnectorAuthType(provider.AuthType)
	if err != nil {
		return nil, fmt.Errorf("%s auth_type 非法: %w", code, err)
	}
	extraAuth, err := marshalStringMap(provider.ExtraAuthParams)
	if err != nil {
		return nil, fmt.Errorf("%s extra_auth_params 非法: %w", code, err)
	}
	extraToken, err := marshalStringMap(provider.ExtraTokenParams)
	if err != nil {
		return nil, fmt.Errorf("%s extra_token_params 非法: %w", code, err)
	}
	clientSecretCipher := ""
	if strings.TrimSpace(provider.ClientSecret) != "" {
		clientSecretCipher, err = s.tokenVault.Seal(provider.ClientSecret)
		if err != nil {
			return nil, err
		}
	}
	setting := &model.ConnectorOAuthProviderSetting{
		Code:               code,
		Name:               firstNonEmpty(provider.Name, code),
		AuthType:           authType,
		ClientID:           strings.TrimSpace(provider.ClientID),
		ClientSecretCipher: clientSecretCipher,
		AuthURL:            strings.TrimSpace(provider.AuthURL),
		TokenURL:           strings.TrimSpace(provider.TokenURL),
		UserInfoURL:        strings.TrimSpace(provider.UserInfoURL),
		Scopes:             strings.Join(cleanScopes(provider.Scopes), " "),
		UsePKCE:            provider.UsePKCE,
		TokenRequestMode:   strings.TrimSpace(provider.TokenRequestMode),
		ClientIDParam:      strings.TrimSpace(provider.ClientIDParam),
		ClientSecretParam:  strings.TrimSpace(provider.ClientSecretParam),
		GrantTypeParam:     strings.TrimSpace(provider.GrantTypeParam),
		CodeParam:          strings.TrimSpace(provider.CodeParam),
		RefreshTokenParam:  strings.TrimSpace(provider.RefreshTokenParam),
		RedirectURIParam:   strings.TrimSpace(provider.RedirectURIParam),
		ExtraAuthParams:    extraAuth,
		ExtraTokenParams:   extraToken,
		ExternalIDField:    strings.TrimSpace(provider.ExternalIDField),
		DisplayNameField:   strings.TrimSpace(provider.DisplayNameField),
		ProviderAccountURL: strings.TrimSpace(provider.ProviderAccountURL),
		LogoURL:            strings.TrimSpace(provider.LogoURL),
		BrandColor:         strings.TrimSpace(provider.BrandColor),
		Enabled:            true,
	}
	setting.CreatedBy = "system"
	setting.UpdatedBy = "system"
	return setting, nil
}

func (s *ConnectorService) providerSettingToMergedInfo(row *model.ConnectorOAuthProviderSetting) (dto.ConnectorOAuthProviderInfo, error) {
	provider, err := s.providerSettingToConfig(row)
	if err != nil {
		return dto.ConnectorOAuthProviderInfo{}, err
	}
	if base, ok := s.oauth.Lookup(row.Code); ok {
		provider = mergeOAuthProvider(base, provider)
	}
	info := providerConfigToInfo(provider)
	applyProviderSettingMetadata(&info, row)
	return info, nil
}

func providerSettingToInfo(row *model.ConnectorOAuthProviderSetting) dto.ConnectorOAuthProviderInfo {
	info := dto.ConnectorOAuthProviderInfo{
		ID:                 row.ID,
		Code:               row.Code,
		Name:               row.Name,
		AuthType:           defaultConnectorAuthType(row.AuthType),
		ClientID:           row.ClientID,
		HasClientSecret:    row.ClientSecretCipher != "",
		AuthURL:            row.AuthURL,
		TokenURL:           row.TokenURL,
		Scopes:             splitScopes(row.Scopes),
		ProviderAccountURL: row.ProviderAccountURL,
		LogoURL:            row.LogoURL,
		BrandColor:         row.BrandColor,
		Enabled:            row.Enabled,
		Active:             row.Enabled && row.ClientID != "" && row.ClientSecretCipher != "" && row.AuthURL != "" && row.TokenURL != "",
		Managed:            true,
		CreatedAt:          formatModelTime(row.CreatedAt),
		UpdatedAt:          formatModelTime(row.UpdatedAt),
	}
	return info
}

func providerConfigToInfo(provider config.ConnectorOAuthProviderConfig) dto.ConnectorOAuthProviderInfo {
	clientID := strings.TrimSpace(firstNonEmpty(provider.ClientID, os.Getenv(provider.ClientIDEnv)))
	hasClientSecret := strings.TrimSpace(firstNonEmpty(provider.ClientSecret, os.Getenv(provider.ClientSecretEnv))) != ""
	return dto.ConnectorOAuthProviderInfo{
		Code:               provider.Code,
		Name:               firstNonEmpty(provider.Name, provider.Code),
		AuthType:           defaultConnectorAuthType(provider.AuthType),
		ClientID:           clientID,
		HasClientSecret:    hasClientSecret,
		AuthURL:            provider.AuthURL,
		TokenURL:           provider.TokenURL,
		Scopes:             cleanScopes(provider.Scopes),
		ProviderAccountURL: provider.ProviderAccountURL,
		LogoURL:            provider.LogoURL,
		BrandColor:         provider.BrandColor,
		Enabled:            true,
		Active:             clientID != "" && hasClientSecret && strings.TrimSpace(provider.AuthURL) != "" && strings.TrimSpace(provider.TokenURL) != "",
		Managed:            false,
	}
}

func applyProviderSettingMetadata(info *dto.ConnectorOAuthProviderInfo, row *model.ConnectorOAuthProviderSetting) {
	info.ID = row.ID
	info.AuthType = defaultConnectorAuthType(firstNonEmpty(info.AuthType, row.AuthType))
	info.Enabled = row.Enabled
	info.Active = row.Enabled && info.ClientID != "" && info.HasClientSecret && strings.TrimSpace(info.AuthURL) != "" && strings.TrimSpace(info.TokenURL) != ""
	info.Managed = true
	info.CreatedAt = formatModelTime(row.CreatedAt)
	info.UpdatedAt = formatModelTime(row.UpdatedAt)
}

func marshalStringMap(values map[string]string) (string, error) {
	if len(values) == 0 {
		return "", nil
	}
	data, err := json.Marshal(values)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func unmarshalStringMap(raw string) (map[string]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var values map[string]string
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil, err
	}
	return values, nil
}

func sortProviderInfos(items []dto.ConnectorOAuthProviderInfo) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].Code < items[j].Code
	})
}

func (s *ConnectorService) requireProviderAdmin(user string) error {
	user = strings.TrimSpace(user)
	if user == "" {
		return fmt.Errorf("未提供用户信息")
	}
	if _, ok := s.admins[user]; ok {
		return nil
	}
	return fmt.Errorf("当前用户无权管理 OAuth provider 配置")
}
