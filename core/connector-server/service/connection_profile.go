package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/config"
)

type oauthConnectionProfile struct {
	DisplayName       string
	ExternalAccountID string
	Metadata          map[string]interface{}
}

func buildOAuthConnectionProfile(ctx context.Context, provider config.ConnectorOAuthProviderConfig, tokenPayload *OAuthTokenPayload) (*oauthConnectionProfile, error) {
	if tokenPayload == nil {
		return nil, fmt.Errorf("OAuth token 为空")
	}
	metadata := map[string]interface{}{
		"oauth":            true,
		"auth_type":        defaultConnectorAuthType(provider.AuthType),
		"provider":         provider.Code,
		"scopes":           splitScopes(tokenPayload.Scopes),
		"last_enriched_at": time.Now().Format(time.RFC3339),
	}
	profile := &dto.ConnectorConnectionProfile{
		Provider:       provider.Code,
		LastEnrichedAt: metadata["last_enriched_at"].(string),
	}
	if provider.ProviderAccountURL != "" {
		profile.AccountURL = provider.ProviderAccountURL
		metadata["provider_account_url"] = provider.ProviderAccountURL
	}

	userInfo, err := fetchProviderUserInfo(ctx, provider, tokenPayload.AccessToken)
	if err != nil {
		metadata["profile_error"] = err.Error()
	}
	if len(userInfo) > 0 {
		metadata["user_info"] = safeOAuthUserInfo(provider.Code, userInfo)
		applyGenericUserInfoProfile(provider, userInfo, profile, metadata)
	}
	adapter := connectorAdapterFor(provider.Code)
	if err := adapter.EnrichOAuthProfile(ctx, provider, tokenPayload, userInfo, profile, metadata); err != nil {
		metadata["profile_adapter_error"] = err.Error()
	}

	if profile.DisplayName == "" {
		profile.DisplayName = firstNonEmpty(profile.WorkspaceName, profile.AccountName)
	}
	metadata["profile"] = profile
	return &oauthConnectionProfile{
		DisplayName:       profile.DisplayName,
		ExternalAccountID: firstNonEmpty(profile.AccountID, profile.WorkspaceID),
		Metadata:          metadata,
	}, nil
}

func applyGenericUserInfoProfile(provider config.ConnectorOAuthProviderConfig, userInfo map[string]interface{}, profile *dto.ConnectorConnectionProfile, metadata map[string]interface{}) {
	externalID, displayName := extractProviderIdentity(provider, userInfo)
	if externalID != "" {
		profile.AccountID = externalID
		metadata["external_id"] = externalID
	}
	if displayName != "" {
		profile.AccountName = displayName
		profile.DisplayName = displayName
		metadata["username"] = displayName
	}
	profile.AvatarURL = firstNonEmpty(profile.AvatarURL, stringField(userInfo, "avatar_url", "picture", "image_url"))
	profile.AccountURL = firstNonEmpty(profile.AccountURL, stringField(userInfo, "html_url", "web_url", "url"))
}

func safeOAuthUserInfo(_ string, userInfo map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{})
	keys := []string{"id", "login", "name", "email", "avatar_url", "html_url", "web_url", "url", "picture", "type", "bot"}
	for _, key := range keys {
		if value, ok := userInfo[key]; ok && value != nil {
			out[key] = value
		}
	}
	return out
}

func oauthTokenRawRoot(raw string) map[string]interface{} {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return nil
	}
	if dataObj, ok := decoded["data"].(map[string]interface{}); ok {
		return dataObj
	}
	return decoded
}
