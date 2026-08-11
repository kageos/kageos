package service

import (
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/kageos/kageos/core/hr-server/model"
	"github.com/kageos/kageos/core/hr-server/repository"
)

const (
	wechatOfficialAttemptTTL = 5 * time.Minute
	wechatOfficialPollAfter  = 2 * time.Second
)

var ErrWechatCallbackUnauthorized = errors.New("wechat callback signature is invalid")

type AuthWechatOfficialService struct {
	providerService *AuthLoginProviderService
	attemptRepo     *repository.AuthWechatLoginAttemptRepository
	oauthService    *AuthOAuthService
	client          *http.Client
	apiBaseURL      string
	qrBaseURL       string

	mu                 sync.Mutex
	cachedAppID        string
	cachedAppSecret    string
	cachedAccessToken  string
	cachedTokenExpires time.Time
}

type WechatLoginAttemptResult struct {
	AttemptToken string
	QRCodeURL    string
	ExpiresAt    time.Time
	PollAfterMS  int
}

type WechatLoginCompleteResult struct {
	Status               string
	Token                string
	RefreshToken         string
	RedirectAfter        string
	RegistrationRequired bool
	RegistrationTicket   string
}

type WechatCallbackInput struct {
	Signature string
	Timestamp string
	Nonce     string
	Echo      string
}

type wechatOfficialEvent struct {
	ToUserName   string `xml:"ToUserName"`
	FromUserName string `xml:"FromUserName"`
	MessageType  string `xml:"MsgType"`
	Event        string `xml:"Event"`
	EventKey     string `xml:"EventKey"`
}

type wechatOfficialAccessTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	ErrorCode   int    `json:"errcode"`
	Error       string `json:"errmsg"`
}

type wechatOfficialQRCodeResponse struct {
	Ticket        string `json:"ticket"`
	ExpireSeconds int    `json:"expire_seconds"`
	ErrorCode     int    `json:"errcode"`
	Error         string `json:"errmsg"`
}

type wechatOfficialSubscriber struct {
	OpenID     string `json:"openid"`
	UnionID    string `json:"unionid"`
	Nickname   string `json:"nickname"`
	AvatarURL  string `json:"headimgurl"`
	Subscribed int    `json:"subscribe"`
	ErrorCode  int    `json:"errcode"`
	Error      string `json:"errmsg"`
}

func NewAuthWechatOfficialService(providerService *AuthLoginProviderService, attemptRepo *repository.AuthWechatLoginAttemptRepository, oauthService *AuthOAuthService) *AuthWechatOfficialService {
	return &AuthWechatOfficialService{
		providerService: providerService,
		attemptRepo:     attemptRepo,
		oauthService:    oauthService,
		client:          &http.Client{Timeout: 12 * time.Second},
		apiBaseURL:      "https://api.weixin.qq.com",
		qrBaseURL:       "https://mp.weixin.qq.com/cgi-bin/showqrcode",
	}
}

func (s *AuthWechatOfficialService) CreateAttempt(ctx context.Context, redirectAfter string) (*WechatLoginAttemptResult, error) {
	values, err := s.runtimeValues()
	if err != nil {
		return nil, err
	}
	attemptToken, err := newOAuthState()
	if err != nil {
		return nil, err
	}
	scene, err := newOAuthState()
	if err != nil {
		return nil, err
	}
	expiresAt := time.Now().Add(wechatOfficialAttemptTTL)
	if err := s.attemptRepo.Create(&model.AuthWechatLoginAttempt{
		TokenHash:     hashWechatLoginValue(attemptToken),
		SceneHash:     hashWechatLoginValue(scene),
		ProviderCode:  ProviderWechatOfficial,
		RedirectAfter: sanitizeRedirectAfter(redirectAfter),
		ExpiresAt:     expiresAt,
	}); err != nil {
		return nil, fmt.Errorf("创建微信扫码登录状态失败: %w", err)
	}

	accessToken, err := s.getAccessToken(ctx, values)
	if err != nil {
		return nil, err
	}
	body := strings.NewReader(fmt.Sprintf(`{"expire_seconds":%d,"action_name":"QR_STR_SCENE","action_info":{"scene":{"scene_str":%q}}}`, int(wechatOfficialAttemptTTL.Seconds()), scene))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.apiBaseURL+"/cgi-bin/qrcode/create?access_token="+url.QueryEscape(accessToken), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	var qr wechatOfficialQRCodeResponse
	if err := s.doJSON(req, &qr); err != nil {
		return nil, fmt.Errorf("创建微信公众号登录二维码失败: %w", err)
	}
	if qr.ErrorCode != 0 || strings.TrimSpace(qr.Ticket) == "" {
		return nil, fmt.Errorf("创建微信公众号登录二维码失败: %d %s", qr.ErrorCode, qr.Error)
	}
	return &WechatLoginAttemptResult{
		AttemptToken: attemptToken,
		QRCodeURL:    s.qrBaseURL + "?ticket=" + url.QueryEscape(qr.Ticket),
		ExpiresAt:    expiresAt,
		PollAfterMS:  int(wechatOfficialPollAfter / time.Millisecond),
	}, nil
}

func (s *AuthWechatOfficialService) CompleteAttempt(ctx context.Context, attemptToken string) (*WechatLoginCompleteResult, error) {
	if _, err := s.runtimeValues(); err != nil {
		return nil, err
	}
	attempt, err := s.attemptRepo.Consume(hashWechatLoginValue(strings.TrimSpace(attemptToken)), time.Now())
	if errors.Is(err, repository.ErrWechatLoginAttemptPending) {
		return &WechatLoginCompleteResult{Status: "pending"}, nil
	}
	if err != nil {
		return nil, err
	}
	result, err := s.oauthService.CompleteExternalLogin(ctx, ExternalPrincipal{
		ProviderCode: ProviderWechatOfficial,
		ExternalID:   attempt.ExternalID,
		Nickname:     attempt.Nickname,
		Avatar:       attempt.Avatar,
	}, ExternalLoginOptions{ShortCode: "wechat", RedirectAfter: attempt.RedirectAfter})
	if err != nil {
		return nil, err
	}
	return &WechatLoginCompleteResult{
		Status:               "complete",
		Token:                result.Token,
		RefreshToken:         result.RefreshToken,
		RedirectAfter:        result.RedirectAfter,
		RegistrationRequired: result.RegistrationRequired,
		RegistrationTicket:   result.RegistrationTicket,
	}, nil
}

func (s *AuthWechatOfficialService) VerifyCallback(input WechatCallbackInput) (string, error) {
	values, err := s.runtimeValues()
	if err != nil || strings.TrimSpace(input.Echo) == "" || !validWechatSignature(values["message_token"], input) {
		return "", ErrWechatCallbackUnauthorized
	}
	return input.Echo, nil
}

func (s *AuthWechatOfficialService) ReceiveEvent(ctx context.Context, input WechatCallbackInput, body io.Reader) error {
	values, err := s.runtimeValues()
	if err != nil || !validWechatSignature(values["message_token"], input) {
		return ErrWechatCallbackUnauthorized
	}
	var event wechatOfficialEvent
	if err := xml.NewDecoder(io.LimitReader(body, 64*1024)).Decode(&event); err != nil {
		return fmt.Errorf("解析微信回调失败: %w", err)
	}
	if !strings.EqualFold(strings.TrimSpace(event.MessageType), "event") {
		return nil
	}
	scene := strings.TrimSpace(event.EventKey)
	if strings.EqualFold(event.Event, "subscribe") {
		scene = strings.TrimPrefix(scene, "qrscene_")
	} else if !strings.EqualFold(event.Event, "SCAN") {
		return nil
	}
	openID := strings.TrimSpace(event.FromUserName)
	if scene == "" || openID == "" {
		return nil
	}
	principal := s.officialPrincipal(ctx, values, openID)
	if err := s.attemptRepo.MarkScanned(hashWechatLoginValue(scene), principal.ExternalID, principal.Nickname, principal.Avatar, time.Now()); err != nil && !errors.Is(err, repository.ErrWechatLoginAttemptInvalid) {
		return fmt.Errorf("更新微信扫码状态失败: %w", err)
	}
	return nil
}

func (s *AuthWechatOfficialService) runtimeValues() (map[string]string, error) {
	config, err := s.providerService.GetEnabledRuntimeConfig(ProviderWechatOfficial)
	if err != nil {
		return nil, fmt.Errorf("微信公众号扫码登录未配置或未启用")
	}
	return config.Values, nil
}

func (s *AuthWechatOfficialService) getAccessToken(ctx context.Context, values map[string]string) (string, error) {
	appID := strings.TrimSpace(values["app_id"])
	appSecret := strings.TrimSpace(values["app_secret"])
	if appID == "" || appSecret == "" {
		return "", fmt.Errorf("微信公众号扫码登录配置不完整")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cachedAccessToken != "" && s.cachedAppID == appID && s.cachedAppSecret == appSecret && time.Now().Before(s.cachedTokenExpires) {
		return s.cachedAccessToken, nil
	}
	query := url.Values{}
	query.Set("grant_type", "client_credential")
	query.Set("appid", appID)
	query.Set("secret", appSecret)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.apiBaseURL+"/cgi-bin/token?"+query.Encode(), nil)
	if err != nil {
		return "", err
	}
	var result wechatOfficialAccessTokenResponse
	if err := s.doJSON(req, &result); err != nil {
		return "", fmt.Errorf("获取微信公众号 access token 失败: %w", err)
	}
	if result.ErrorCode != 0 || result.AccessToken == "" {
		return "", fmt.Errorf("获取微信公众号 access token 失败: %d %s", result.ErrorCode, result.Error)
	}
	expiresIn := result.ExpiresIn
	if expiresIn <= 300 {
		expiresIn = 600
	}
	s.cachedAppID = appID
	s.cachedAppSecret = appSecret
	s.cachedAccessToken = result.AccessToken
	s.cachedTokenExpires = time.Now().Add(time.Duration(expiresIn-300) * time.Second)
	return result.AccessToken, nil
}

func (s *AuthWechatOfficialService) officialPrincipal(ctx context.Context, values map[string]string, openID string) ExternalPrincipal {
	appID := strings.TrimSpace(values["app_id"])
	principal := ExternalPrincipal{ProviderCode: ProviderWechatOfficial, ExternalID: appID + ":" + openID, Nickname: "微信用户"}
	accessToken, err := s.getAccessToken(ctx, values)
	if err != nil {
		return principal
	}
	query := url.Values{}
	query.Set("access_token", accessToken)
	query.Set("openid", openID)
	query.Set("lang", "zh_CN")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.apiBaseURL+"/cgi-bin/user/info?"+query.Encode(), nil)
	if err != nil {
		return principal
	}
	var subscriber wechatOfficialSubscriber
	if s.doJSON(req, &subscriber) != nil || subscriber.ErrorCode != 0 {
		return principal
	}
	if nickname := strings.TrimSpace(subscriber.Nickname); nickname != "" {
		principal.Nickname = nickname
	}
	principal.Avatar = strings.TrimSpace(subscriber.AvatarURL)
	return principal
}

func (s *AuthWechatOfficialService) doJSON(req *http.Request, output interface{}) error {
	resp, err := s.client.Do(req)
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

func validWechatSignature(messageToken string, input WechatCallbackInput) bool {
	if strings.TrimSpace(messageToken) == "" || strings.TrimSpace(input.Signature) == "" || strings.TrimSpace(input.Timestamp) == "" || strings.TrimSpace(input.Nonce) == "" {
		return false
	}
	parts := []string{messageToken, input.Timestamp, input.Nonce}
	sort.Strings(parts)
	sum := sha1.Sum([]byte(strings.Join(parts, "")))
	expected := hex.EncodeToString(sum[:])
	return subtle.ConstantTimeCompare([]byte(expected), []byte(strings.ToLower(strings.TrimSpace(input.Signature)))) == 1
}

func hashWechatLoginValue(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
