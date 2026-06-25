package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kageos/kageos/core/hr-server/model"
	"github.com/kageos/kageos/pkg/logger"
	"gorm.io/gorm"
)

const defaultExternalDepartmentFullPath = "/org/unassigned"

type ExternalPrincipal struct {
	ProviderCode      string
	ExternalID        string
	Username          string
	PreferredUsername string
	Email             string
	EmailVerified     bool
	Nickname          string
	Avatar            string
}

type ExternalLoginOptions struct {
	ShortCode          string
	DefaultCompanyCode string
	RedirectAfter      string
}

type ExternalLoginResult struct {
	User                 *model.User
	Token                string
	RefreshToken         string
	RedirectAfter        string
	RegistrationRequired bool
	RegistrationTicket   string
}

func (s *AuthOAuthService) CompleteExternalLogin(ctx context.Context, principal ExternalPrincipal, options ExternalLoginOptions) (*ExternalLoginResult, error) {
	_ = ctx
	principal = normalizeExternalPrincipal(principal)
	if principal.ProviderCode == "" || principal.ExternalID == "" {
		return nil, fmt.Errorf("外部用户信息不完整")
	}

	user, found, err := s.resolveExistingUserForPrincipal(principal)
	if err != nil {
		return nil, err
	}
	if !found {
		intent, err := s.createExternalRegistrationIntent(principal, options)
		if err != nil {
			return nil, err
		}
		return &ExternalLoginResult{
			RedirectAfter:        sanitizeRedirectAfter(options.RedirectAfter),
			RegistrationRequired: true,
			RegistrationTicket:   intent.Ticket,
		}, nil
	}

	accessToken, refreshToken, err := s.authService.IssueTokensForUser(user, false)
	if err != nil {
		return nil, err
	}
	return &ExternalLoginResult{
		User:          user,
		Token:         accessToken,
		RefreshToken:  refreshToken,
		RedirectAfter: sanitizeRedirectAfter(options.RedirectAfter),
	}, nil
}

func (s *AuthOAuthService) resolveExistingUserForPrincipal(principal ExternalPrincipal) (*model.User, bool, error) {
	principal = normalizeExternalPrincipal(principal)
	if principal.ProviderCode == "" || principal.ExternalID == "" {
		return nil, false, fmt.Errorf("外部用户信息不完整")
	}

	identity, err := s.identityRepo.GetByProviderSubject(principal.ProviderCode, principal.ExternalID)
	if err != nil {
		return nil, false, err
	}
	if identity != nil {
		user, err := s.userRepo.GetUserByID(identity.UserID)
		if err != nil {
			return nil, false, fmt.Errorf("已绑定用户不存在，请联系管理员")
		}
		s.refreshExternalIdentity(identity, principal)
		return user, true, nil
	}

	return nil, false, nil
}

func (s *AuthOAuthService) createExternalRegistrationIntent(principal ExternalPrincipal, options ExternalLoginOptions) (*model.AuthOAuthRegistrationIntent, error) {
	principal = normalizeExternalPrincipal(principal)
	if principal.ProviderCode == "" || principal.ExternalID == "" {
		return nil, fmt.Errorf("外部用户信息不完整")
	}
	ticket, err := newOAuthState()
	if err != nil {
		return nil, err
	}
	suggestions, err := s.suggestExternalUserCodes(principal, options.ShortCode)
	if err != nil {
		return nil, err
	}
	if len(suggestions) == 0 {
		return nil, fmt.Errorf("生成用户 code 建议失败")
	}

	companyCode := strings.TrimSpace(options.DefaultCompanyCode)
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
	nickname := strings.TrimSpace(principal.Nickname)
	if nickname == "" {
		nickname = suggestions[0]
	}
	intent := &model.AuthOAuthRegistrationIntent{
		Ticket:              ticket,
		ProviderCode:        principal.ProviderCode,
		ExternalID:          principal.ExternalID,
		Email:               principal.Email,
		EmailVerified:       userEmailVerified(principal.Email, principal.EmailVerified),
		Nickname:            nickname,
		Avatar:              principal.Avatar,
		SuggestedCode:       suggestions[0],
		CodeSuggestionsJSON: string(suggestionsJSON),
		RedirectAfter:       sanitizeRedirectAfter(options.RedirectAfter),
		CompanyCode:         companyCode,
		ExpiresAt:           time.Now().Add(oauthRegistrationIntentTTL),
	}
	if err := s.registrationIntentRepo.Create(intent); err != nil {
		return nil, fmt.Errorf("创建授权注册确认失败: %w", err)
	}
	return intent, nil
}

func (s *AuthOAuthService) suggestExternalUserCodes(principal ExternalPrincipal, shortCode string) ([]string, error) {
	principal = normalizeExternalPrincipal(principal)
	shortCode = normalizeProviderCode(shortCode)
	if shortCode == "" {
		shortCode = "external"
	}

	seen := map[string]struct{}{}
	out := make([]string, 0, 4)
	addAvailable := func(raw string) error {
		if strings.TrimSpace(raw) == "" {
			return nil
		}
		base := NormalizeUserCodeCandidate(raw)
		candidates := []string{
			base,
			trimUserCodeForSuffix(base, "_"+shortCode) + "_" + shortCode,
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
	for _, raw := range []string{principal.Username, principal.PreferredUsername, principal.Email, principal.Nickname, "user"} {
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

func (s *AuthOAuthService) refreshExternalIdentity(identity *model.AuthExternalIdentity, principal ExternalPrincipal) {
	principal = normalizeExternalPrincipal(principal)
	changed := false
	if principal.Email != "" && identity.Email != principal.Email {
		identity.Email = principal.Email
		changed = true
	}
	if principal.Avatar != "" && identity.Avatar != principal.Avatar {
		identity.Avatar = principal.Avatar
		changed = true
	}
	if principal.Nickname != "" && identity.Nickname != principal.Nickname {
		identity.Nickname = principal.Nickname
		changed = true
	}
	if changed {
		if err := s.identityRepo.Update(identity); err != nil {
			logger.Warnf(nil, "[AuthExternalLogin] 刷新外部身份失败: %v", err)
		}
	}
}

func normalizeExternalPrincipal(principal ExternalPrincipal) ExternalPrincipal {
	principal.ProviderCode = normalizeProviderCode(principal.ProviderCode)
	principal.ExternalID = strings.TrimSpace(principal.ExternalID)
	principal.Username = strings.TrimSpace(principal.Username)
	principal.PreferredUsername = strings.TrimSpace(principal.PreferredUsername)
	principal.Email = strings.ToLower(strings.TrimSpace(principal.Email))
	principal.Nickname = strings.TrimSpace(principal.Nickname)
	principal.Avatar = strings.TrimSpace(principal.Avatar)
	return principal
}

func externalPrincipalFromOAuthProfile(profile *OAuthProfile) ExternalPrincipal {
	if profile == nil {
		return ExternalPrincipal{}
	}
	return ExternalPrincipal{
		ProviderCode:      profile.ProviderCode,
		ExternalID:        profile.ExternalID,
		PreferredUsername: profile.PreferredUsername,
		Email:             profile.Email,
		EmailVerified:     profile.EmailVerified,
		Nickname:          profile.Nickname,
		Avatar:            profile.Avatar,
	}
}
