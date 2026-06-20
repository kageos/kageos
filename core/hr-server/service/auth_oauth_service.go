package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kageos/kageos/core/hr-server/model"
	"github.com/kageos/kageos/core/hr-server/repository"
	"github.com/kageos/kageos/pkg/logger"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

const (
	oauthStateTTL              = 10 * time.Minute
	oauthRegistrationIntentTTL = 20 * time.Minute
)

type AuthOAuthService struct {
	authService            *AuthService
	providerService        *AuthLoginProviderService
	stateRepo              *repository.AuthOAuthStateRepository
	registrationIntentRepo *repository.AuthOAuthRegistrationIntentRepository
	identityRepo           *repository.AuthExternalIdentityRepository
	userRepo               *repository.UserRepository
}

type OAuthLoginResult struct {
	User                 *model.User
	Token                string
	RefreshToken         string
	RedirectAfter        string
	RegistrationRequired bool
	RegistrationTicket   string
}

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

type oauthProfile struct {
	ProviderCode      string
	ExternalID        string
	Email             string
	EmailVerified     bool
	PreferredUsername string
	Nickname          string
	Avatar            string
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
	conf, err := s.oauth2Config(providerCode, runtimeConfig.Values)
	if err != nil {
		return "", err
	}
	state, err := newOAuthState()
	if err != nil {
		return "", err
	}
	if err := s.stateRepo.Create(&model.AuthOAuthState{
		State:         state,
		ProviderCode:  providerCode,
		RedirectAfter: sanitizeRedirectAfter(redirectAfter),
		ExpiresAt:     time.Now().Add(oauthStateTTL),
	}); err != nil {
		return "", fmt.Errorf("创建授权状态失败: %w", err)
	}

	opts := []oauth2.AuthCodeOption{oauth2.SetAuthURLParam("prompt", "select_account")}
	if providerCode == ProviderGitHubOAuth {
		opts = nil
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
	conf, err := s.oauth2Config(providerCode, runtimeConfig.Values)
	if err != nil {
		return nil, err
	}
	token, err := conf.Exchange(ctx, strings.TrimSpace(code))
	if err != nil {
		return nil, fmt.Errorf("换取授权令牌失败: %w", err)
	}
	profile, err := s.fetchProfile(ctx, providerCode, conf.Client(ctx, token))
	if err != nil {
		return nil, err
	}
	user, found, err := s.resolveExistingUserForProfile(profile)
	if err != nil {
		return nil, err
	}
	if !found {
		intent, err := s.createRegistrationIntent(profile, runtimeConfig.Values, oauthState.RedirectAfter)
		if err != nil {
			return nil, err
		}
		return &OAuthLoginResult{
			RedirectAfter:        oauthState.RedirectAfter,
			RegistrationRequired: true,
			RegistrationTicket:   intent.Ticket,
		}, nil
	}
	accessToken, refreshToken, err := s.authService.IssueTokensForUser(user, false)
	if err != nil {
		return nil, err
	}
	return &OAuthLoginResult{
		User:          user,
		Token:         accessToken,
		RefreshToken:  refreshToken,
		RedirectAfter: oauthState.RedirectAfter,
	}, nil
}

func (s *AuthOAuthService) GetRegistrationIntent(ticket string) (*OAuthRegistrationIntentView, error) {
	intent, err := s.activeRegistrationIntent(ticket)
	if err != nil {
		return nil, err
	}
	view := oauthRegistrationIntentView(intent)
	if shouldRefreshOAuthCodeSuggestions(view.CodeSuggestions) {
		suggestions, err := s.suggestUserCodes(&oauthProfile{
			ProviderCode:  intent.ProviderCode,
			ExternalID:    intent.ExternalID,
			Email:         intent.Email,
			EmailVerified: intent.EmailVerified,
			Nickname:      intent.Nickname,
			Avatar:        intent.Avatar,
		})
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

	companyCode := strings.TrimSpace(intent.CompanyCode)
	if companyCode == "" {
		companyCode = model.DefaultCompanyCode
	}
	if s.authService != nil && s.authService.companyRepo != nil {
		if _, err := s.authService.companyRepo.GetCompanyByCode(companyCode); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, fmt.Errorf("企业代码 %s 不存在，请联系系统管理员", companyCode)
			}
			return nil, fmt.Errorf("检查企业代码失败: %w", err)
		}
	}

	user := &model.User{
		Username:           username,
		Email:              strings.ToLower(strings.TrimSpace(intent.Email)),
		CompanyCode:        companyCode,
		RegisterType:       oauthRegisterType(intent.ProviderCode),
		Status:             "active",
		EmailVerified:      userEmailVerified(intent.Email, intent.EmailVerified),
		CreatedBy:          intent.ProviderCode,
		ThirdPartyID:       intent.ExternalID,
		Avatar:             intent.Avatar,
		Nickname:           nickname,
		DepartmentFullPath: "/org/unassigned",
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
	clientID := strings.TrimSpace(values["client_id"])
	clientSecret := strings.TrimSpace(values["client_secret"])
	redirectURL := strings.TrimSpace(values["redirect_url"])
	if clientID == "" || clientSecret == "" || redirectURL == "" {
		return nil, fmt.Errorf("授权配置不完整")
	}

	switch providerCode {
	case ProviderGoogleOAuth:
		return &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       []string{"openid", "email", "profile"},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://accounts.google.com/o/oauth2/v2/auth",
				TokenURL: "https://oauth2.googleapis.com/token",
			},
		}, nil
	case ProviderGitHubOAuth:
		scopes := strings.Fields(values["scopes"])
		if len(scopes) == 0 {
			scopes = []string{"read:user", "user:email"}
		}
		return &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       scopes,
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://github.com/login/oauth/authorize",
				TokenURL: "https://github.com/login/oauth/access_token",
			},
		}, nil
	default:
		return nil, fmt.Errorf("暂不支持该授权登录方式")
	}
}

func (s *AuthOAuthService) fetchProfile(ctx context.Context, providerCode string, client *http.Client) (*oauthProfile, error) {
	switch providerCode {
	case ProviderGoogleOAuth:
		return fetchGoogleProfile(ctx, client)
	case ProviderGitHubOAuth:
		return fetchGitHubProfile(ctx, client)
	default:
		return nil, fmt.Errorf("暂不支持该授权登录方式")
	}
}

func (s *AuthOAuthService) resolveExistingUserForProfile(profile *oauthProfile) (*model.User, bool, error) {
	if profile == nil || profile.ExternalID == "" {
		return nil, false, fmt.Errorf("授权用户信息不完整")
	}

	identity, err := s.identityRepo.GetByProviderSubject(profile.ProviderCode, profile.ExternalID)
	if err != nil {
		return nil, false, err
	}
	if identity != nil {
		user, err := s.userRepo.GetUserByID(identity.UserID)
		if err != nil {
			return nil, false, fmt.Errorf("已绑定用户不存在，请联系管理员")
		}
		s.refreshIdentity(identity, profile)
		return user, true, nil
	}

	return nil, false, nil
}

func (s *AuthOAuthService) createRegistrationIntent(profile *oauthProfile, values map[string]string, redirectAfter string) (*model.AuthOAuthRegistrationIntent, error) {
	email := strings.ToLower(strings.TrimSpace(profile.Email))
	ticket, err := newOAuthState()
	if err != nil {
		return nil, err
	}
	suggestions, err := s.suggestUserCodes(profile)
	if err != nil {
		return nil, err
	}
	if len(suggestions) == 0 {
		return nil, fmt.Errorf("生成用户 code 建议失败")
	}
	companyCode := strings.TrimSpace(values["default_company_code"])
	if companyCode == "" {
		companyCode = model.DefaultCompanyCode
	}
	if s.authService != nil && s.authService.companyRepo != nil {
		if _, err := s.authService.companyRepo.GetCompanyByCode(companyCode); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, fmt.Errorf("生成授权注册失败：企业代码 %s 不存在", companyCode)
			}
			return nil, fmt.Errorf("检查企业代码失败: %w", err)
		}
	}
	suggestionsJSON, err := json.Marshal(suggestions)
	if err != nil {
		return nil, err
	}
	nickname := strings.TrimSpace(profile.Nickname)
	if nickname == "" {
		nickname = suggestions[0]
	}
	intent := &model.AuthOAuthRegistrationIntent{
		Ticket:              ticket,
		ProviderCode:        profile.ProviderCode,
		ExternalID:          profile.ExternalID,
		Email:               email,
		EmailVerified:       email != "" && profile.EmailVerified,
		Nickname:            nickname,
		Avatar:              profile.Avatar,
		SuggestedCode:       suggestions[0],
		CodeSuggestionsJSON: string(suggestionsJSON),
		RedirectAfter:       sanitizeRedirectAfter(redirectAfter),
		CompanyCode:         companyCode,
		ExpiresAt:           time.Now().Add(oauthRegistrationIntentTTL),
	}
	if err := s.registrationIntentRepo.Create(intent); err != nil {
		return nil, fmt.Errorf("创建授权注册确认失败: %w", err)
	}
	return intent, nil
}

func (s *AuthOAuthService) suggestUserCodes(profile *oauthProfile) ([]string, error) {
	seen := map[string]struct{}{}
	out := make([]string, 0, 4)
	addAvailable := func(raw string) error {
		if strings.TrimSpace(raw) == "" {
			return nil
		}
		base := NormalizeUserCodeCandidate(raw)
		candidates := []string{
			base,
			trimUserCodeForSuffix(base, "_"+oauthProviderShortCode(profile.ProviderCode)) + "_" + oauthProviderShortCode(profile.ProviderCode),
		}
		for i := 2; i <= 50; i++ {
			suffix := fmt.Sprintf("_%02d", i)
			candidates = append(candidates, trimUserCodeForSuffix(base, suffix)+suffix)
		}
		for _, candidate := range candidates {
			if len(out) >= 4 {
				return nil
			}
			if _, ok := seen[candidate]; ok {
				continue
			}
			seen[candidate] = struct{}{}
			if err := ValidateUserCode(candidate); err != nil {
				continue
			}
			if available, err := s.userCodeAvailable(candidate); err != nil {
				return err
			} else if available {
				out = append(out, candidate)
			}
		}
		return nil
	}
	for _, raw := range []string{profile.PreferredUsername, profile.Email, profile.Nickname, "user"} {
		if err := addAvailable(raw); err != nil {
			return nil, err
		}
		if len(out) >= 4 {
			break
		}
	}
	return out, nil
}

func (s *AuthOAuthService) userCodeAvailable(code string) (bool, error) {
	if _, err := s.userRepo.GetUserByUsername(code); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return true, nil
		}
		return false, fmt.Errorf("检查用户 code 失败: %w", err)
	}
	return false, nil
}

func oauthRegisterType(providerCode string) string {
	switch providerCode {
	case ProviderGoogleOAuth:
		return "google"
	case ProviderGitHubOAuth:
		return "github"
	default:
		return "oauth"
	}
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
	switch providerCode {
	case ProviderGoogleOAuth:
		return "Google"
	case ProviderGitHubOAuth:
		return "GitHub"
	default:
		return providerCode
	}
}

func oauthProviderShortCode(providerCode string) string {
	switch providerCode {
	case ProviderGoogleOAuth:
		return "google"
	case ProviderGitHubOAuth:
		return "github"
	default:
		return "oauth"
	}
}

func (s *AuthOAuthService) refreshIdentity(identity *model.AuthExternalIdentity, profile *oauthProfile) {
	changed := false
	if email := strings.ToLower(strings.TrimSpace(profile.Email)); email != "" && identity.Email != email {
		identity.Email = email
		changed = true
	}
	if profile.Avatar != "" && identity.Avatar != profile.Avatar {
		identity.Avatar = profile.Avatar
		changed = true
	}
	if profile.Nickname != "" && identity.Nickname != profile.Nickname {
		identity.Nickname = profile.Nickname
		changed = true
	}
	if changed {
		if err := s.identityRepo.Update(identity); err != nil {
			logger.Warnf(nil, "[AuthOAuthService] 刷新授权身份失败: %v", err)
		}
	}
}

func fetchGoogleProfile(ctx context.Context, client *http.Client) (*oauthProfile, error) {
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
	return &oauthProfile{
		ProviderCode:  ProviderGoogleOAuth,
		ExternalID:    payload.Sub,
		Email:         strings.ToLower(strings.TrimSpace(payload.Email)),
		EmailVerified: payload.EmailVerified,
		Nickname:      payload.Name,
		Avatar:        payload.Picture,
	}, nil
}

func fetchGitHubProfile(ctx context.Context, client *http.Client) (*oauthProfile, error) {
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
	return &oauthProfile{
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
	return json.NewDecoder(resp.Body).Decode(out)
}

func oauthProviderCode(alias string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(alias)) {
	case "google", ProviderGoogleOAuth:
		return ProviderGoogleOAuth, nil
	case "github", ProviderGitHubOAuth:
		return ProviderGitHubOAuth, nil
	default:
		return "", fmt.Errorf("暂不支持该授权登录方式")
	}
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

func fillUserProfileFromOAuth(user *model.User, profile *oauthProfile) bool {
	changed := false
	if user.Avatar == "" && profile.Avatar != "" {
		user.Avatar = profile.Avatar
		changed = true
	}
	if user.Nickname == "" && profile.Nickname != "" {
		user.Nickname = profile.Nickname
		changed = true
	}
	if !user.EmailVerified && profile.EmailVerified && strings.EqualFold(user.Email, profile.Email) {
		user.EmailVerified = true
		changed = true
	}
	return changed
}
