package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kageos/kageos/core/hr-server/model"
	"github.com/kageos/kageos/core/hr-server/repository"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

const (
	oauthStateTTL                = 10 * time.Minute
	oauthRegistrationIntentTTL   = 20 * time.Minute
	maxOAuthProfileResponseBytes = 1 << 20
)

type AuthOAuthService struct {
	authService            *AuthService
	providerService        *AuthLoginProviderService
	stateRepo              *repository.AuthOAuthStateRepository
	registrationIntentRepo *repository.AuthOAuthRegistrationIntentRepository
	identityRepo           *repository.AuthExternalIdentityRepository
	userRepo               *repository.UserRepository
}

type OAuthLoginResult = ExternalLoginResult

type OAuthRegistrationIntentView struct {
	Ticket          string
	ProviderCode    string
	ProviderName    string
	Email           string
	Nickname        string
	Avatar          string
	SuggestedCode   string
	CodeSuggestions []string
	RedirectAfter   string
	ExpiresAt       time.Time
}

type OAuthRegistrationConfirmResult struct {
	User          *model.User
	Token         string
	RefreshToken  string
	RedirectAfter string
}

func NewAuthOAuthService(
	authService *AuthService,
	providerService *AuthLoginProviderService,
	stateRepo *repository.AuthOAuthStateRepository,
	registrationIntentRepo *repository.AuthOAuthRegistrationIntentRepository,
	identityRepo *repository.AuthExternalIdentityRepository,
	userRepo *repository.UserRepository,
) *AuthOAuthService {
	return &AuthOAuthService{
		authService:            authService,
		providerService:        providerService,
		stateRepo:              stateRepo,
		registrationIntentRepo: registrationIntentRepo,
		identityRepo:           identityRepo,
		userRepo:               userRepo,
	}
}

func (s *AuthOAuthService) StartAuthorize(ctx context.Context, providerAlias, redirectAfter string) (string, error) {
	providerCode, err := oauthProviderCode(providerAlias)
	if err != nil {
		return "", err
	}
	runtimeConfig, err := s.providerService.GetEnabledRuntimeConfig(providerCode)
	if err != nil {
		return "", err
	}
	state, err := newOAuthState()
	if err != nil {
		return "", err
	}
	factory, _ := GetOAuthProvider(providerCode)
	pkceVerifier := ""
	if factory.UsePKCE {
		pkceVerifier = oauth2.GenerateVerifier()
	}
	if err := s.stateRepo.Create(&model.AuthOAuthState{
		State:         state,
		ProviderCode:  providerCode,
		RedirectAfter: sanitizeRedirectAfter(redirectAfter),
		PKCEVerifier:  pkceVerifier,
		ExpiresAt:     time.Now().Add(oauthStateTTL),
	}); err != nil {
		return "", fmt.Errorf("创建授权状态失败: %w", err)
	}
	if factory.BuildAuthorizeURL != nil {
		return factory.BuildAuthorizeURL(runtimeConfig.Values, state)
	}
	conf, err := s.oauth2Config(providerCode, runtimeConfig.Values)
	if err != nil {
		return "", err
	}

	var opts []oauth2.AuthCodeOption
	if factory, ok := GetOAuthProvider(providerCode); ok && factory.AuthCodeOptions != nil {
		opts = factory.AuthCodeOptions(runtimeConfig.Values)
	}
	if pkceVerifier != "" {
		opts = append(opts, oauth2.S256ChallengeOption(pkceVerifier))
	}
	return conf.AuthCodeURL(state, opts...), nil
}

func (s *AuthOAuthService) FinishCallback(ctx context.Context, providerAlias, state, code, providerError string) (*OAuthLoginResult, error) {
	if strings.TrimSpace(providerError) != "" {
		return nil, fmt.Errorf("授权平台返回错误: %s", providerError)
	}
	if strings.TrimSpace(state) == "" || strings.TrimSpace(code) == "" {
		return nil, fmt.Errorf("授权回调缺少 code 或 state")
	}
	providerCode, err := oauthProviderCode(providerAlias)
	if err != nil {
		return nil, err
	}
	oauthState, err := s.stateRepo.Consume(strings.TrimSpace(state), providerCode)
	if err != nil {
		return nil, fmt.Errorf("授权状态无效或已过期")
	}
	runtimeConfig, err := s.providerService.GetEnabledRuntimeConfig(providerCode)
	if err != nil {
		return nil, err
	}
	factory, ok := GetOAuthProvider(providerCode)
	if !ok {
		return nil, fmt.Errorf("暂不支持该授权登录方式")
	}
	var profile *OAuthProfile
	if factory.ExchangeProfile != nil {
		profile, err = factory.ExchangeProfile(ctx, runtimeConfig.Values, strings.TrimSpace(code))
	} else {
		conf, configErr := s.oauth2Config(providerCode, runtimeConfig.Values)
		if configErr != nil {
			return nil, configErr
		}
		var exchangeOptions []oauth2.AuthCodeOption
		if oauthState.PKCEVerifier != "" {
			exchangeOptions = append(exchangeOptions, oauth2.VerifierOption(oauthState.PKCEVerifier))
		}
		token, exchangeErr := conf.Exchange(ctx, strings.TrimSpace(code), exchangeOptions...)
		if exchangeErr != nil {
			return nil, fmt.Errorf("换取授权令牌失败: %w", exchangeErr)
		}
		profile, err = s.fetchProfile(ctx, providerCode, conf.Client(ctx, token))
	}
	if err != nil {
		return nil, err
	}
	if profile != nil && profile.ProviderCode == "" {
		profile.ProviderCode = providerCode
	}
	return s.CompleteExternalLogin(ctx, externalPrincipalFromOAuthProfile(profile), ExternalLoginOptions{
		ShortCode:     oauthProviderShortCode(providerCode),
		RedirectAfter: oauthState.RedirectAfter,
	})
}

func (s *AuthOAuthService) GetRegistrationIntent(ticket string) (*OAuthRegistrationIntentView, error) {
	intent, err := s.activeRegistrationIntent(ticket)
	if err != nil {
		return nil, err
	}
	view := oauthRegistrationIntentView(intent)
	if shouldRefreshOAuthCodeSuggestions(view.CodeSuggestions) {
		suggestions, err := s.suggestExternalUserCodes(ExternalPrincipal{
			ProviderCode:  intent.ProviderCode,
			ExternalID:    intent.ExternalID,
			Email:         intent.Email,
			EmailVerified: intent.EmailVerified,
			Nickname:      intent.Nickname,
			Avatar:        intent.Avatar,
		}, oauthProviderShortCode(intent.ProviderCode))
		if err == nil && len(suggestions) > 0 {
			view.SuggestedCode = suggestions[0]
			view.CodeSuggestions = suggestions
		}
	}
	return view, nil
}

func (s *AuthOAuthService) ConfirmRegistration(ticket, username, nickname string) (*OAuthRegistrationConfirmResult, error) {
	intent, err := s.activeRegistrationIntent(ticket)
	if err != nil {
		return nil, err
	}
	username = strings.ToLower(strings.TrimSpace(username))
	if err := ValidateUserCode(username); err != nil {
		return nil, err
	}
	nickname = strings.TrimSpace(nickname)
	if nickname == "" {
		nickname = intent.Nickname
	}
	if nickname == "" {
		nickname = username
	}
	nickname = truncateRunes(nickname, 100)

	user := &model.User{
		Username:           username,
		Email:              strings.ToLower(strings.TrimSpace(intent.Email)),
		RegisterType:       oauthRegisterType(intent.ProviderCode),
		Status:             "active",
		EmailVerified:      userEmailVerified(intent.Email, intent.EmailVerified),
		CreatedBy:          intent.ProviderCode,
		ThirdPartyID:       intent.ExternalID,
		Avatar:             intent.Avatar,
		Nickname:           nickname,
		DepartmentFullPath: defaultExternalDepartmentFullPath,
		Type:               model.UserTypeNormal,
	}
	identity := &model.AuthExternalIdentity{
		ProviderCode: intent.ProviderCode,
		ExternalID:   intent.ExternalID,
		Email:        user.Email,
		Avatar:       intent.Avatar,
		Nickname:     nickname,
	}
	completedIntent, err := s.registrationIntentRepo.Complete(strings.TrimSpace(ticket), user, identity)
	if err != nil {
		return nil, oauthRegistrationCompleteError(err)
	}
	accessToken, refreshToken, err := s.authService.IssueTokensForUser(user, false)
	if err != nil {
		return nil, err
	}
	return &OAuthRegistrationConfirmResult{
		User:          user,
		Token:         accessToken,
		RefreshToken:  refreshToken,
		RedirectAfter: completedIntent.RedirectAfter,
	}, nil
}

func (s *AuthOAuthService) activeRegistrationIntent(ticket string) (*model.AuthOAuthRegistrationIntent, error) {
	ticket = strings.TrimSpace(ticket)
	if ticket == "" {
		return nil, fmt.Errorf("授权注册确认不存在或已失效")
	}
	intent, err := s.registrationIntentRepo.GetByTicket(ticket)
	if err != nil {
		return nil, fmt.Errorf("读取授权注册确认失败: %w", err)
	}
	if intent == nil || intent.UsedAt != nil || time.Now().After(intent.ExpiresAt) {
		return nil, fmt.Errorf("授权注册确认不存在或已失效，请重新授权登录")
	}
	return intent, nil
}

func (s *AuthOAuthService) oauth2Config(providerCode string, values map[string]string) (*oauth2.Config, error) {
	factory, ok := GetOAuthProvider(providerCode)
	if !ok || factory.OAuth2Config == nil {
		return nil, fmt.Errorf("暂不支持该授权登录方式")
	}
	return factory.OAuth2Config(values)
}

func (s *AuthOAuthService) fetchProfile(ctx context.Context, providerCode string, client *http.Client) (*OAuthProfile, error) {
	factory, ok := GetOAuthProvider(providerCode)
	if !ok || factory.FetchProfile == nil {
		return nil, fmt.Errorf("暂不支持该授权登录方式")
	}
	profile, err := factory.FetchProfile(ctx, client)
	if err != nil {
		return nil, err
	}
	if profile != nil && profile.ProviderCode == "" {
		profile.ProviderCode = providerCode
	}
	return profile, nil
}

func oauthRegisterType(providerCode string) string {
	if factory, ok := GetOAuthProvider(providerCode); ok && strings.TrimSpace(factory.RegisterType) != "" {
		return strings.TrimSpace(factory.RegisterType)
	}
	if normalizeProviderCode(providerCode) == ProviderWechatOfficial {
		return "wechat"
	}
	return "oauth"
}

func userEmailVerified(email string, verified bool) bool {
	return strings.TrimSpace(email) != "" && verified
}

func oauthRegistrationIntentView(intent *model.AuthOAuthRegistrationIntent) *OAuthRegistrationIntentView {
	return &OAuthRegistrationIntentView{
		Ticket:          intent.Ticket,
		ProviderCode:    intent.ProviderCode,
		ProviderName:    oauthProviderDisplayName(intent.ProviderCode),
		Email:           intent.Email,
		Nickname:        intent.Nickname,
		Avatar:          intent.Avatar,
		SuggestedCode:   intent.SuggestedCode,
		CodeSuggestions: parseOAuthCodeSuggestions(intent.CodeSuggestionsJSON),
		RedirectAfter:   intent.RedirectAfter,
		ExpiresAt:       intent.ExpiresAt,
	}
}

func parseOAuthCodeSuggestions(raw string) []string {
	var suggestions []string
	if strings.TrimSpace(raw) == "" {
		return suggestions
	}
	_ = json.Unmarshal([]byte(raw), &suggestions)
	return suggestions
}

func shouldRefreshOAuthCodeSuggestions(suggestions []string) bool {
	if len(suggestions) == 0 {
		return true
	}
	for _, suggestion := range suggestions {
		if strings.TrimSpace(suggestion) == "u_0" {
			return true
		}
	}
	return false
}

func oauthRegistrationCompleteError(err error) error {
	switch {
	case errors.Is(err, repository.ErrOAuthRegistrationUsernameTaken):
		return fmt.Errorf("用户 code 已被占用，请换一个")
	case errors.Is(err, repository.ErrOAuthRegistrationEmailTaken):
		return fmt.Errorf("该邮箱已被注册，请重新授权登录以绑定已有账号")
	case errors.Is(err, repository.ErrOAuthRegistrationExternalIDBound):
		return fmt.Errorf("该授权账号已绑定，请重新登录")
	case errors.Is(err, repository.ErrOAuthRegistrationIntentUsed),
		errors.Is(err, repository.ErrOAuthRegistrationIntentExpired),
		errors.Is(err, gorm.ErrRecordNotFound):
		return fmt.Errorf("授权注册确认不存在或已失效，请重新授权登录")
	default:
		return fmt.Errorf("确认授权注册失败: %w", err)
	}
}

func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

func oauthProviderDisplayName(providerCode string) string {
	if factory, ok := GetOAuthProvider(providerCode); ok && strings.TrimSpace(factory.DisplayName) != "" {
		return strings.TrimSpace(factory.DisplayName)
	}
	if seed, ok := LookupAuthProviderSeed(providerCode); ok && strings.TrimSpace(seed.Name) != "" {
		return strings.TrimSpace(seed.Name)
	}
	return providerCode
}

func oauthProviderShortCode(providerCode string) string {
	if factory, ok := GetOAuthProvider(providerCode); ok && strings.TrimSpace(factory.ShortCode) != "" {
		return strings.TrimSpace(factory.ShortCode)
	}
	if normalizeProviderCode(providerCode) == ProviderWechatOfficial {
		return "wechat"
	}
	return "oauth"
}

func fetchGoogleProfile(ctx context.Context, client *http.Client) (*OAuthProfile, error) {
	var payload struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
	}
	if err := getJSON(ctx, client, "https://openidconnect.googleapis.com/v1/userinfo", &payload); err != nil {
		return nil, fmt.Errorf("获取 Google 用户信息失败: %w", err)
	}
	return &OAuthProfile{
		ProviderCode:  ProviderGoogleOAuth,
		ExternalID:    payload.Sub,
		Email:         strings.ToLower(strings.TrimSpace(payload.Email)),
		EmailVerified: payload.EmailVerified,
		Nickname:      payload.Name,
		Avatar:        payload.Picture,
	}, nil
}

func fetchGitHubProfile(ctx context.Context, client *http.Client) (*OAuthProfile, error) {
	var userPayload struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := getJSON(ctx, client, "https://api.github.com/user", &userPayload); err != nil {
		return nil, fmt.Errorf("获取 GitHub 用户信息失败: %w", err)
	}

	email := strings.ToLower(strings.TrimSpace(userPayload.Email))
	emailVerified := email != ""
	if email == "" {
		var emails []struct {
			Email    string `json:"email"`
			Primary  bool   `json:"primary"`
			Verified bool   `json:"verified"`
		}
		if err := getJSON(ctx, client, "https://api.github.com/user/emails", &emails); err != nil {
			return nil, fmt.Errorf("获取 GitHub 邮箱失败: %w", err)
		}
		for _, item := range emails {
			if item.Primary && item.Verified {
				email = strings.ToLower(strings.TrimSpace(item.Email))
				emailVerified = true
				break
			}
		}
	}
	nickname := strings.TrimSpace(userPayload.Name)
	if nickname == "" {
		nickname = strings.TrimSpace(userPayload.Login)
	}
	return &OAuthProfile{
		ProviderCode:      ProviderGitHubOAuth,
		ExternalID:        fmt.Sprintf("%d", userPayload.ID),
		Email:             email,
		EmailVerified:     emailVerified,
		PreferredUsername: userPayload.Login,
		Nickname:          nickname,
		Avatar:            userPayload.AvatarURL,
	}, nil
}

func getJSON(ctx context.Context, client *http.Client, endpoint string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Kageos OAuth")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxOAuthProfileResponseBytes+1))
	if err != nil {
		return err
	}
	if len(data) > maxOAuthProfileResponseBytes {
		return fmt.Errorf("OAuth profile response exceeds %d MiB limit", maxOAuthProfileResponseBytes>>20)
	}
	return json.Unmarshal(data, out)
}

func oauthProviderCode(alias string) (string, error) {
	if code, ok := LookupOAuthProviderCode(alias); ok {
		return code, nil
	}
	return "", fmt.Errorf("暂不支持该授权登录方式")
}

func newOAuthState() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func sanitizeRedirectAfter(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") {
		return "/workspace"
	}
	if parsed, err := url.Parse(raw); err != nil || parsed.IsAbs() {
		return "/workspace"
	}
	return raw
}
