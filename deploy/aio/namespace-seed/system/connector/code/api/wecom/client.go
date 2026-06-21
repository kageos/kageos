package wecom

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/types"
	"gorm.io/gorm"
)

const (
	wecomAPIBaseURL      = "https://qyapi.weixin.qq.com"
	wecomTokenRefreshGap = 5 * time.Minute
)

var wecomHTTPClient = &http.Client{Timeout: 12 * time.Second}

type wecomTokenResp struct {
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

type wecomBaseResp struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

func wecomDB(ctx *app.Context) (*gorm.DB, error) {
	db := ctx.GetGormDB()
	if db == nil {
		return nil, fmt.Errorf("数据库连接失败")
	}
	if err := db.AutoMigrate(&WeComConfig{}); err != nil {
		return nil, fmt.Errorf("初始化企业微信配置表失败: %w", err)
	}
	return db, nil
}

func loadWeComConfig(ctx *app.Context, configID int) (*WeComConfig, error) {
	db, err := wecomDB(ctx)
	if err != nil {
		return nil, err
	}
	var cfg WeComConfig
	query := db.Model(&WeComConfig{})
	if configID > 0 {
		query = query.Where("id = ?", configID)
	} else {
		query = query.Where("enabled = ?", true).Order("id ASC")
	}
	if err := query.First(&cfg).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			if configID > 0 {
				return nil, fmt.Errorf("未找到企业微信配置 ID=%d", configID)
			}
			return nil, fmt.Errorf("还没有可用的企业微信配置，请先打开“企业微信配置”填写 corp_id、agent_id、corp_secret")
		}
		return nil, err
	}
	if !cfg.Enabled {
		return nil, fmt.Errorf("企业微信配置“%s”已停用", cfg.Name)
	}
	return &cfg, nil
}

func ensureWeComAccessToken(ctx *app.Context, cfg *WeComConfig, forceRefresh bool) (string, error) {
	if cfg == nil {
		return "", fmt.Errorf("企业微信配置不能为空")
	}
	now := time.Now()
	if !forceRefresh && strings.TrimSpace(cfg.AccessTokenCipher) != "" && cfg.TokenExpiresAt.Time().After(now.Add(wecomTokenRefreshGap)) {
		token, err := decryptWeComSecret(cfg.AccessTokenCipher)
		if err == nil && strings.TrimSpace(token) != "" {
			return token, nil
		}
	}
	secret, err := decryptWeComSecret(cfg.CorpSecretCipher)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(secret) == "" {
		return "", fmt.Errorf("企业微信应用 Secret 未配置")
	}
	tokenResp, err := fetchWeComAccessToken(cfg.CorpID, secret)
	if err != nil {
		updateWeComConfigStatus(ctx, cfg.ID, "失败", err.Error(), time.Time{}, "")
		return "", err
	}
	expiresAt := now.Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	tokenCipher, err := encryptWeComSecret(tokenResp.AccessToken)
	if err != nil {
		return "", err
	}
	updateWeComConfigStatus(ctx, cfg.ID, "正常", "access_token 获取成功", expiresAt, tokenCipher)
	cfg.AccessTokenCipher = tokenCipher
	cfg.TokenExpiresAt = types.Time(expiresAt)
	cfg.LastStatus = "正常"
	cfg.LastMessage = "access_token 获取成功"
	return tokenResp.AccessToken, nil
}

func fetchWeComAccessToken(corpID, corpSecret string) (*wecomTokenResp, error) {
	corpID = strings.TrimSpace(corpID)
	corpSecret = strings.TrimSpace(corpSecret)
	if corpID == "" || corpSecret == "" {
		return nil, fmt.Errorf("corp_id 和 corp_secret 不能为空")
	}
	values := url.Values{}
	values.Set("corpid", corpID)
	values.Set("corpsecret", corpSecret)
	endpoint := wecomAPIBaseURL + "/cgi-bin/gettoken?" + values.Encode()
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	httpResp, err := wecomHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求企业微信 access_token 失败: %w", err)
	}
	defer httpResp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(httpResp.Body, 1<<20))
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return nil, fmt.Errorf("企业微信 gettoken HTTP %d: %s", httpResp.StatusCode, string(body))
	}
	var parsed wecomTokenResp
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("解析企业微信 gettoken 响应失败: %w", err)
	}
	if parsed.ErrCode != 0 {
		return nil, fmt.Errorf("企业微信 gettoken 失败 [%d]: %s", parsed.ErrCode, humanWeComError(parsed.ErrCode, parsed.ErrMsg))
	}
	if strings.TrimSpace(parsed.AccessToken) == "" {
		return nil, fmt.Errorf("企业微信 gettoken 未返回 access_token")
	}
	if parsed.ExpiresIn <= 0 {
		parsed.ExpiresIn = 7200
	}
	return &parsed, nil
}

func postWeComAPI(ctx *app.Context, cfg *WeComConfig, apiPath string, reqBody interface{}, out interface{}) error {
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		token, err := ensureWeComAccessToken(ctx, cfg, attempt > 0)
		if err != nil {
			return err
		}
		err = postWeComAPIWithToken(apiPath, token, reqBody, out)
		if err == nil {
			return nil
		}
		lastErr = err
		if !isWeComTokenError(err) {
			break
		}
	}
	return lastErr
}

func postWeComAPIWithToken(apiPath, token string, reqBody interface{}, out interface{}) error {
	apiPath = "/" + strings.TrimLeft(apiPath, "/")
	values := url.Values{}
	values.Set("access_token", token)
	endpoint := wecomAPIBaseURL + apiPath + "?" + values.Encode()
	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("序列化企业微信请求失败: %w", err)
	}
	httpReq, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json; charset=utf-8")
	httpResp, err := wecomHTTPClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("请求企业微信 API 失败: %w", err)
	}
	defer httpResp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(httpResp.Body, 2<<20))
	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return fmt.Errorf("企业微信 API HTTP %d: %s", httpResp.StatusCode, string(respBody))
	}
	if out == nil {
		var base wecomBaseResp
		if err := json.Unmarshal(respBody, &base); err != nil {
			return fmt.Errorf("解析企业微信 API 响应失败: %w", err)
		}
		if base.ErrCode != 0 {
			return newWeComAPIError(base.ErrCode, base.ErrMsg)
		}
		return nil
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("解析企业微信 API 响应失败: %w", err)
	}
	if base, ok := out.(interface {
		Base() wecomBaseResp
	}); ok {
		resp := base.Base()
		if resp.ErrCode != 0 {
			return newWeComAPIError(resp.ErrCode, resp.ErrMsg)
		}
	}
	return nil
}

type wecomAPIError struct {
	Code int
	Msg  string
}

func (e *wecomAPIError) Error() string {
	return fmt.Sprintf("企业微信 API 失败 [%d]: %s", e.Code, humanWeComError(e.Code, e.Msg))
}

func newWeComAPIError(code int, msg string) error {
	return &wecomAPIError{Code: code, Msg: msg}
}

func isWeComTokenError(err error) bool {
	apiErr, ok := err.(*wecomAPIError)
	if !ok {
		return false
	}
	switch apiErr.Code {
	case 40001, 40014, 42001:
		return true
	default:
		return false
	}
}

func humanWeComError(code int, msg string) string {
	msg = strings.TrimSpace(msg)
	hint := ""
	switch code {
	case -1:
		hint = "企业微信系统繁忙，请稍后重试"
	case 40001:
		hint = "凭证无效，请检查应用 Secret 是否正确"
	case 40013:
		hint = "corp_id 不合法，请检查企业 ID"
	case 40014:
		hint = "access_token 无效，系统会尝试重新获取"
	case 42001:
		hint = "access_token 已过期"
	case 48002:
		hint = "API 接口权限不足，请到企业微信管理后台检查应用权限"
	case 60020:
		hint = "访问 IP 不在企业微信可信 IP 列表"
	case 81013:
		hint = "目标用户、部门或标签为空"
	}
	if hint == "" {
		if strings.Contains(strings.ToLower(msg), "not allow to access from your ip") {
			hint = "访问 IP 不在企业微信可信 IP 列表"
		} else {
			hint = "请根据企业微信返回码检查应用配置、可见范围和接口权限"
		}
	}
	if msg == "" {
		return hint
	}
	return hint + "；原始错误: " + msg
}

func updateWeComConfigStatus(ctx *app.Context, id int, status, message string, expiresAt time.Time, tokenCipher string) {
	db, err := wecomDB(ctx)
	if err != nil {
		return
	}
	updates := map[string]interface{}{
		"last_status":  status,
		"last_message": message,
	}
	if !expiresAt.IsZero() {
		updates["token_expires_at"] = types.Time(expiresAt)
	}
	if tokenCipher != "" {
		updates["access_token_cipher"] = tokenCipher
	}
	_ = db.Model(&WeComConfig{}).Where("id = ?", id).Updates(updates).Error
}

func maskWeComSecret(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) <= 8 {
		return strings.Repeat("*", len(value))
	}
	return value[:4] + strings.Repeat("*", len(value)-8) + value[len(value)-4:]
}
