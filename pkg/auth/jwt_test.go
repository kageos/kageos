package auth

import (
	"strings"
	"testing"
	"time"

	appconfig "github.com/kageos/kageos/pkg/config"
)

func TestGenerateRefreshTokenWithContextExpiresAtUsesRequestedExpiry(t *testing.T) {
	service := &JWTService{
		config: &appconfig.JWTConfig{
			Secret:             "test-secret",
			Issuer:             "test-issuer",
			AccessTokenExpire:  60,
			RefreshTokenExpire: 3600,
		},
	}
	expiresAt := time.Now().Add(2 * time.Hour).Truncate(time.Second)

	token, err := service.GenerateRefreshTokenWithContextExpiresAt(UserTokenContext{
		UserID:             42,
		Username:           "alice",
		Email:              "alice@example.com",
		CompanyCode:        "acme",
		DepartmentFullPath: "/org/engineering",
		LeaderUsername:     "lead",
	}, expiresAt)
	if err != nil {
		t.Fatal(err)
	}

	claims, err := service.ValidateToken(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.ExpiresAt == nil {
		t.Fatal("refresh token missing exp")
	}
	if !claims.ExpiresAt.Time.Equal(expiresAt) {
		t.Fatalf("refresh exp = %s, want %s", claims.ExpiresAt.Time, expiresAt)
	}
	if claims.DepartmentFullPath == nil || *claims.DepartmentFullPath != "/org/engineering" {
		t.Fatalf("department claim = %#v, want /org/engineering", claims.DepartmentFullPath)
	}
	if claims.LeaderUsername == nil || *claims.LeaderUsername != "lead" {
		t.Fatalf("leader claim = %#v, want lead", claims.LeaderUsername)
	}
}

func TestOpenAPITokenUsesUsernameSubject(t *testing.T) {
	service := &JWTService{
		config: &appconfig.JWTConfig{
			Secret: "test-secret",
			Issuer: "test-issuer",
		},
	}
	token, err := service.GenerateOpenAPITokenWithContext(UserTokenContext{
		UserID:   42,
		Username: "alice",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := service.ValidateOpenAPIToken(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Subject != "openapi:alice" {
		t.Fatalf("subject = %q, want openapi:alice", claims.Subject)
	}
	if claims.UserID != 42 || claims.Username != "alice" {
		t.Fatalf("claims = %#v, want both user ID and username", claims)
	}
}

func TestValidateOpenAPITokenRejectsAccessToken(t *testing.T) {
	service := &JWTService{
		config: &appconfig.JWTConfig{
			Secret:            "test-secret",
			Issuer:            "test-issuer",
			AccessTokenExpire: 60,
		},
	}
	token, err := service.GenerateAccessToken(42, "alice", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ValidateOpenAPIToken(token); err == nil || !strings.Contains(err.Error(), "不是 OpenAPI Token") {
		t.Fatalf("access token should be rejected as OpenAPI Token, got %v", err)
	}
}

func TestTokenValidatorsRejectWrongPurpose(t *testing.T) {
	service := &JWTService{
		config: &appconfig.JWTConfig{
			Secret:             "test-secret",
			Issuer:             "test-issuer",
			AccessTokenExpire:  60,
			RefreshTokenExpire: 60,
		},
	}
	accessToken, err := service.GenerateAccessToken(42, "alice", "")
	if err != nil {
		t.Fatal(err)
	}
	refreshToken, err := service.GenerateRefreshToken(42, "alice", "")
	if err != nil {
		t.Fatal(err)
	}
	openAPIToken, err := service.GenerateOpenAPITokenWithContext(UserTokenContext{
		UserID:   42,
		Username: "alice",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := service.ValidateAccessToken(accessToken); err != nil {
		t.Fatalf("access token should validate: %v", err)
	}
	for name, token := range map[string]string{
		"refresh": refreshToken,
		"openapi": openAPIToken,
	} {
		if _, err := service.ValidateAccessToken(token); err == nil {
			t.Fatalf("%s token must not validate as an access token", name)
		}
	}
	if _, err := service.ValidateRefreshToken(refreshToken); err != nil {
		t.Fatalf("refresh token should validate: %v", err)
	}
	if _, err := service.ValidateRefreshToken(accessToken); err == nil {
		t.Fatal("access token must not validate as a refresh token")
	}
}
