package openapitoken

import (
	"path/filepath"
	"testing"

	"github.com/kageos/kageos/pkg/auth"
	"github.com/kageos/kageos/pkg/config"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestValidateRequiresOpenAPITokenSubject(t *testing.T) {
	restore := useOpenAPISubjectTestJWTConfig(t)
	defer restore()
	user := auth.UserTokenContext{
		UserID:   42,
		Username: "alice",
		Email:    "alice@example.com",
	}
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "subject-token.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	restoreDB := useOpenAPITestDB(t, database)
	defer restoreDB()
	created, err := Create(CreateInput{
		OwnerUserID:   user.UserID,
		OwnerUsername: user.Username,
		OwnerEmail:    user.Email,
		Name:          "subject test",
	})
	if err != nil {
		t.Fatal(err)
	}
	openAPIToken := created.Secret
	jwtService := auth.NewJWTService()
	accessToken, err := jwtService.GenerateAccessTokenWithContext(user)
	if err != nil {
		t.Fatal(err)
	}
	refreshToken, err := jwtService.GenerateRefreshTokenWithContext(user)
	if err != nil {
		t.Fatal(err)
	}

	principal, err := Validate(openAPIToken, "203.0.113.10", "subject-test")
	if err != nil {
		t.Fatalf("validate real OpenAPI token: %v", err)
	}
	if principal == nil || principal.Username != "alice" {
		t.Fatalf("OpenAPI principal = %#v", principal)
	}

	for name, token := range map[string]string{
		"access":  accessToken,
		"refresh": refreshToken,
	} {
		t.Run(name, func(t *testing.T) {
			if principal, err := Validate(token, "203.0.113.10", "subject-test"); err == nil {
				t.Fatalf("non-OpenAPI token was accepted: %#v", principal)
			}
		})
	}
}

func TestValidateFailsClosedWithoutOpenAPITokenStore(t *testing.T) {
	restore := useOpenAPISubjectTestJWTConfig(t)
	defer restore()

	dbMu.Lock()
	previous := db
	db = nil
	dbMu.Unlock()
	defer func() {
		dbMu.Lock()
		db = previous
		dbMu.Unlock()
	}()

	token, err := auth.NewJWTService().GenerateOpenAPITokenWithContext(auth.UserTokenContext{
		UserID: 42, Username: "alice", Email: "alice@example.com",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if principal, err := Validate(token, "203.0.113.10", "storeless-test"); err == nil {
		t.Fatalf("store-less OpenAPI token was accepted: %#v", principal)
	}
}

func useOpenAPISubjectTestJWTConfig(t *testing.T) func() {
	t.Helper()
	global := config.GetGlobalSharedConfig()
	previous := global.JWT
	global.JWT = config.JWTConfig{
		Secret:             "openapi-subject-test-jwt-secret",
		Issuer:             "openapi-subject-test",
		AccessTokenExpire:  300,
		RefreshTokenExpire: 300,
	}
	return func() {
		global.JWT = previous
	}
}
