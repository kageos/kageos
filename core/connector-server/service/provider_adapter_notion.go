package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/config"
)

const notionVersionHeader = "2026-03-11"

type notionProviderAdapter struct {
	defaultProviderAdapter
}

func (notionProviderAdapter) Code() string {
	return "notion"
}

func (notionProviderAdapter) Capabilities() dto.ConnectorProviderCapabilities {
	return dto.ConnectorProviderCapabilities{
		OAuthSupported:           true,
		ProxySupported:           true,
		ProfileSupported:         true,
		ResourceSummarySupported: true,
	}
}

func (notionProviderAdapter) ProxyBaseURL() string {
	return "https://api.notion.com"
}

func (notionProviderAdapter) UseAccessTypeOffline() bool {
	return false
}

func (notionProviderAdapter) DecorateAPIRequest(req *http.Request) {
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Notion-Version", notionVersionHeader)
}

func (notionProviderAdapter) EnrichOAuthProfile(ctx context.Context, provider config.ConnectorOAuthProviderConfig, tokenPayload *OAuthTokenPayload, userInfo map[string]interface{}, profile *dto.ConnectorConnectionProfile, metadata map[string]interface{}) error {
	tokenRoot := oauthTokenRawRoot(tokenPayload.RawResponse)
	applyNotionTokenProfile(tokenRoot, profile, metadata)
	summary, err := fetchNotionResourceSummary(ctx, provider, tokenPayload.AccessToken)
	if err != nil {
		metadata["resource_summary_error"] = err.Error()
		return nil
	}
	if summary != nil {
		profile.ResourceSummary = summary
	}
	return nil
}

func applyNotionTokenProfile(tokenRoot map[string]interface{}, profile *dto.ConnectorConnectionProfile, metadata map[string]interface{}) {
	workspaceID := stringField(tokenRoot, "workspace_id")
	workspaceName := stringField(tokenRoot, "workspace_name")
	workspaceIcon := stringField(tokenRoot, "workspace_icon")
	botID := stringField(tokenRoot, "bot_id")
	if workspaceID != "" {
		profile.WorkspaceID = workspaceID
		metadata["workspace_id"] = workspaceID
	}
	if workspaceName != "" {
		profile.WorkspaceName = workspaceName
		profile.DisplayName = workspaceName
		metadata["workspace_name"] = workspaceName
	}
	if workspaceIcon != "" {
		profile.WorkspaceIcon = workspaceIcon
		metadata["workspace_icon"] = workspaceIcon
	}
	if botID != "" {
		profile.AccountID = botID
		metadata["bot_id"] = botID
	}
	if owner, ok := tokenRoot["owner"].(map[string]interface{}); ok {
		applyNotionOwnerProfile(owner, profile, metadata)
	}
}

func applyNotionOwnerProfile(owner map[string]interface{}, profile *dto.ConnectorConnectionProfile, metadata map[string]interface{}) {
	userObj, ok := owner["user"].(map[string]interface{})
	if !ok {
		return
	}
	ownerID := stringField(userObj, "id")
	ownerName := stringField(userObj, "name")
	ownerAvatar := stringField(userObj, "avatar_url")
	if ownerID != "" {
		metadata["owner_user_id"] = ownerID
	}
	if ownerName != "" {
		profile.AccountName = ownerName
		metadata["owner_name"] = ownerName
	}
	if ownerAvatar != "" {
		profile.AvatarURL = ownerAvatar
		metadata["owner_avatar_url"] = ownerAvatar
	}
}

func fetchNotionResourceSummary(ctx context.Context, provider config.ConnectorOAuthProviderConfig, accessToken string) (*dto.ConnectorResourceSummary, error) {
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return nil, nil
	}
	searchURL := notionAPIURL(provider, "/v1/search")
	body := []byte(`{"page_size":20}`)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, searchURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	decorateProviderAPIRequest("notion", req)
	resp, err := connectorOAuthHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("notion search returned %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	var raw struct {
		Results []map[string]interface{} `json:"results"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	summary := &dto.ConnectorResourceSummary{}
	for _, item := range raw.Results {
		switch stringField(item, "object") {
		case "page":
			summary.PageCount++
		case "database", "data_source":
			summary.DatabaseCount++
		}
		if len(summary.Samples) < 5 {
			if title := notionObjectTitle(item); title != "" {
				summary.Samples = append(summary.Samples, title)
			}
		}
	}
	if summary.PageCount == 0 && summary.DatabaseCount == 0 && len(summary.Samples) == 0 {
		return nil, nil
	}
	return summary, nil
}

func notionAPIURL(provider config.ConnectorOAuthProviderConfig, apiPath string) string {
	tokenURL := strings.TrimSpace(provider.TokenURL)
	if tokenURL == "" {
		return "https://api.notion.com" + apiPath
	}
	idx := strings.Index(tokenURL, "/v1/")
	if idx < 0 {
		return "https://api.notion.com" + apiPath
	}
	return strings.TrimRight(tokenURL[:idx], "/") + apiPath
}

func notionObjectTitle(item map[string]interface{}) string {
	if title := richTextPlainText(item["title"]); title != "" {
		return title
	}
	properties, _ := item["properties"].(map[string]interface{})
	for _, raw := range properties {
		prop, ok := raw.(map[string]interface{})
		if !ok || stringField(prop, "type") != "title" {
			continue
		}
		if title := richTextPlainText(prop["title"]); title != "" {
			return title
		}
	}
	return ""
}

func richTextPlainText(value interface{}) string {
	items, ok := value.([]interface{})
	if !ok {
		return oauthValueString(value)
	}
	parts := make([]string, 0, len(items))
	for _, item := range items {
		obj, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if text := stringField(obj, "plain_text"); text != "" {
			parts = append(parts, text)
		}
	}
	return strings.TrimSpace(strings.Join(parts, ""))
}
