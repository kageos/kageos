package publicshare

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/config"
)

const (
	AnonymousTokenHeader = "X-Public-Anonymous-Token"
	tokenTTL             = 365 * 24 * time.Hour
)

type AnonymousClaims struct {
	Type      string `json:"typ"`
	SessionID string `json:"sid"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

func IssueAnonymousToken() (string, *AnonymousClaims, error) {
	sid, err := randomSessionID()
	if err != nil {
		return "", nil, err
	}
	now := time.Now()
	claims := &AnonymousClaims{
		Type:      "public_anonymous",
		SessionID: sid,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(tokenTTL).Unix(),
	}
	token, err := signClaims(claims)
	return token, claims, err
}

func ValidateOrIssueAnonymousToken(token string) (string, *AnonymousClaims, error) {
	claims, err := ValidateAnonymousToken(token)
	if err == nil {
		return token, claims, nil
	}
	return IssueAnonymousToken()
}

func ValidateAnonymousToken(token string) (*AnonymousClaims, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, fmt.Errorf("anonymous token is empty")
	}
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return nil, fmt.Errorf("anonymous token format invalid")
	}
	secret := secretBytes()
	if len(secret) == 0 {
		return nil, fmt.Errorf("public anonymous token secret is empty")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("anonymous token payload invalid: %w", err)
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("anonymous token signature invalid: %w", err)
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write(payload)
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return nil, fmt.Errorf("anonymous token signature mismatch")
	}
	var claims AnonymousClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("anonymous token claims invalid: %w", err)
	}
	if claims.Type != "public_anonymous" || claims.SessionID == "" {
		return nil, fmt.Errorf("anonymous token claims invalid")
	}
	if claims.ExpiresAt > 0 && time.Now().Unix() > claims.ExpiresAt {
		return nil, fmt.Errorf("anonymous token expired")
	}
	return &claims, nil
}

func DeriveActorID(tenantUser, app, shareID, sessionID string) string {
	msg := strings.Join([]string{tenantUser, app, shareID, sessionID}, "|")
	mac := hmac.New(sha256.New, secretBytes())
	mac.Write([]byte(msg))
	return "guest_anon_" + hex.EncodeToString(mac.Sum(nil))[:20]
}

func HashValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	mac := hmac.New(sha256.New, secretBytes())
	mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}

func signClaims(claims *AnonymousClaims) (string, error) {
	secret := secretBytes()
	if len(secret) == 0 {
		return "", fmt.Errorf("public anonymous token secret is empty")
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write(payload)
	return base64.RawURLEncoding.EncodeToString(payload) + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func randomSessionID() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func secretBytes() []byte {
	if v := strings.TrimSpace(os.Getenv("PUBLIC_ANONYMOUS_TOKEN_SECRET")); v != "" {
		return []byte(v)
	}
	return []byte(strings.TrimSpace(config.GetGlobalSharedConfig().JWT.Secret))
}
