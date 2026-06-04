package service

import (
	"context"
	"net/http"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/config"
)

type githubProviderAdapter struct {
	defaultProviderAdapter
}

func (githubProviderAdapter) Code() string {
	return "github"
}

func (githubProviderAdapter) ProxyBaseURL() string {
	return "https://api.github.com"
}

func (githubProviderAdapter) UseAccessTypeOffline() bool {
	return true
}

func (githubProviderAdapter) DecorateAPIRequest(req *http.Request) {
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
}

func (githubProviderAdapter) MissingScopes(granted, required []string) []string {
	return defaultMissingScopes(granted, required, githubScopeGranted)
}

func (githubProviderAdapter) EnrichOAuthProfile(ctx context.Context, provider config.ConnectorOAuthProviderConfig, tokenPayload *OAuthTokenPayload, userInfo map[string]interface{}, profile *dto.ConnectorConnectionProfile, metadata map[string]interface{}) error {
	if len(userInfo) == 0 {
		return nil
	}
	login := stringField(userInfo, "login")
	if login != "" {
		profile.AccountName = login
		profile.DisplayName = login
		metadata["username"] = login
	}
	profile.AvatarURL = firstNonEmpty(profile.AvatarURL, stringField(userInfo, "avatar_url"))
	profile.AccountURL = firstNonEmpty(profile.AccountURL, stringField(userInfo, "html_url"))
	if profile.AvatarURL != "" {
		metadata["avatar_url"] = profile.AvatarURL
	}
	if profile.AccountURL != "" {
		metadata["provider_account_url"] = profile.AccountURL
	}
	return nil
}

func githubScopeGranted(granted map[string]struct{}, required string) bool {
	if defaultScopeGranted(granted, required) {
		return true
	}
	if _, ok := granted["repo"]; ok {
		switch required {
		case "public_repo", "repo:status", "repo_deployment", "repo:invite", "security_events", "admin:repo_hook", "write:repo_hook", "read:repo_hook":
			return true
		}
	}
	if _, ok := granted["user"]; ok {
		switch required {
		case "read:user", "user:email", "user:follow":
			return true
		}
	}
	if _, ok := granted["admin:org"]; ok {
		switch required {
		case "write:org", "read:org":
			return true
		}
	}
	if _, ok := granted["write:org"]; ok && required == "read:org" {
		return true
	}
	return false
}
