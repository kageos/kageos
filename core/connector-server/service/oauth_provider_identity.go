package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/kageos/kageos/pkg/config"
)

func fetchProviderUserInfo(ctx context.Context, provider config.ConnectorOAuthProviderConfig, accessToken string) (map[string]interface{}, error) {
	if strings.TrimSpace(provider.UserInfoURL) == "" || strings.TrimSpace(accessToken) == "" {
		return nil, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, provider.UserInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	decorateProviderAPIRequest(provider.Code, req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("oauth userinfo endpoint returned %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	if dataObj, ok := raw["data"].(map[string]interface{}); ok {
		return dataObj, nil
	}
	return raw, nil
}

func extractProviderIdentity(provider config.ConnectorOAuthProviderConfig, userInfo map[string]interface{}) (externalID, displayName string) {
	if len(userInfo) == 0 {
		return "", ""
	}
	externalID = stringFieldByPath(userInfo, provider.ExternalIDField)
	if externalID == "" {
		externalID = stringField(userInfo, "id", "sub", "openid", "open_id", "unionid", "union_id", "login", "email")
	}
	displayName = stringFieldByPath(userInfo, provider.DisplayNameField)
	if displayName == "" {
		displayName = stringField(userInfo, "name", "display_name", "displayName", "login", "email", "nickname")
	}
	return externalID, displayName
}

func stringFieldByPath(raw map[string]interface{}, path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	current := interface{}(raw)
	for _, part := range strings.Split(path, ".") {
		obj, ok := current.(map[string]interface{})
		if !ok {
			return ""
		}
		next, ok := obj[part]
		if !ok || next == nil {
			return ""
		}
		current = next
	}
	return oauthValueString(current)
}

func oauthValueString(value interface{}) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(typed)
	case float64:
		return strconv.FormatInt(int64(typed), 10)
	case float32:
		return strconv.FormatInt(int64(typed), 10)
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case int32:
		return strconv.FormatInt(int64(typed), 10)
	case json.Number:
		return typed.String()
	case []interface{}:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			if value := oauthValueString(item); value != "" {
				values = append(values, value)
			}
		}
		return strings.Join(values, " ")
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}
