package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const wechatOpenAPIBaseURL = "https://api.weixin.qq.com"

type wechatOpenTokenResponse struct {
	AccessToken string `json:"access_token"`
	OpenID      string `json:"openid"`
	UnionID     string `json:"unionid"`
	ErrorCode   int    `json:"errcode"`
	Error       string `json:"errmsg"`
}

type wechatOpenUserInfoResponse struct {
	OpenID    string `json:"openid"`
	UnionID   string `json:"unionid"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"headimgurl"`
	ErrorCode int    `json:"errcode"`
	Error     string `json:"errmsg"`
}

func buildWechatOpenAuthorizeURL(values map[string]string, state string) (string, error) {
	appID, _, redirectURL, err := wechatOpenClientValues(values)
	if err != nil {
		return "", err
	}
	query := url.Values{}
	query.Set("appid", appID)
	query.Set("redirect_uri", redirectURL)
	query.Set("response_type", "code")
	query.Set("scope", "snsapi_login")
	query.Set("state", state)
	return "https://open.weixin.qq.com/connect/qrconnect?" + query.Encode() + "#wechat_redirect", nil
}

func exchangeWechatOpenProfile(ctx context.Context, values map[string]string, code string) (*OAuthProfile, error) {
	appID, appSecret, _, err := wechatOpenClientValues(values)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 12 * time.Second}
	tokenQuery := url.Values{}
	tokenQuery.Set("appid", appID)
	tokenQuery.Set("secret", appSecret)
	tokenQuery.Set("code", strings.TrimSpace(code))
	tokenQuery.Set("grant_type", "authorization_code")
	var token wechatOpenTokenResponse
	if err := getWechatJSON(ctx, client, wechatOpenAPIBaseURL+"/sns/oauth2/access_token?"+tokenQuery.Encode(), &token); err != nil {
		return nil, fmt.Errorf("换取微信授权令牌失败: %w", err)
	}
	if token.ErrorCode != 0 || token.AccessToken == "" || token.OpenID == "" {
		return nil, fmt.Errorf("换取微信授权令牌失败: %d %s", token.ErrorCode, token.Error)
	}

	profileQuery := url.Values{}
	profileQuery.Set("access_token", token.AccessToken)
	profileQuery.Set("openid", token.OpenID)
	profileQuery.Set("lang", "zh_CN")
	var userInfo wechatOpenUserInfoResponse
	if err := getWechatJSON(ctx, client, wechatOpenAPIBaseURL+"/sns/userinfo?"+profileQuery.Encode(), &userInfo); err != nil {
		return nil, fmt.Errorf("获取微信用户信息失败: %w", err)
	}
	if userInfo.ErrorCode != 0 {
		return nil, fmt.Errorf("获取微信用户信息失败: %d %s", userInfo.ErrorCode, userInfo.Error)
	}
	openID := firstNonEmptyWechatValue(userInfo.OpenID, token.OpenID)
	unionID := firstNonEmptyWechatValue(userInfo.UnionID, token.UnionID)
	externalID := "openid:" + appID + ":" + openID
	if unionID != "" {
		externalID = "unionid:" + unionID
	}
	return &OAuthProfile{
		ProviderCode: ProviderWechatOpenOAuth,
		ExternalID:   externalID,
		Nickname:     strings.TrimSpace(userInfo.Nickname),
		Avatar:       strings.TrimSpace(userInfo.AvatarURL),
	}, nil
}

func wechatOpenClientValues(values map[string]string) (string, string, string, error) {
	appID := strings.TrimSpace(values["app_id"])
	appSecret := strings.TrimSpace(values["app_secret"])
	redirectURL := strings.TrimSpace(values["redirect_url"])
	if appID == "" || appSecret == "" || redirectURL == "" {
		return "", "", "", fmt.Errorf("微信开放平台授权配置不完整")
	}
	parsed, err := url.Parse(redirectURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", "", "", fmt.Errorf("微信开放平台回调地址无效")
	}
	return appID, appSecret, redirectURL, nil
}

func getWechatJSON(ctx context.Context, client *http.Client, endpoint string, output interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("微信接口请求失败")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64*1024))
		return fmt.Errorf("微信接口返回 HTTP %d", resp.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, maxOAuthProfileResponseBytes)).Decode(output)
}

func firstNonEmptyWechatValue(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
