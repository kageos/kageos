package auth

import (
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
