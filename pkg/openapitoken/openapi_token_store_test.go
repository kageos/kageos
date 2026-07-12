package openapitoken

import (
	"path/filepath"
	"testing"

	"github.com/kageos/kageos/pkg/auth"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestValidateConfiguredStoreRejectsUnknownAndRevokedTokens(t *testing.T) {
	restoreJWT := useOpenAPISubjectTestJWTConfig(t)
	defer restoreJWT()
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "openapi-token-test.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	restoreDB := useOpenAPITestDB(t, database)
	defer restoreDB()

	created, err := Create(CreateInput{
		OwnerUserID:   42,
		OwnerUsername: "alice",
		OwnerEmail:    "alice@example.com",
		Name:          "agent automation",
	})
	if err != nil {
		t.Fatal(err)
	}
	principal, err := Validate(created.Secret, "203.0.113.10", "store-test")
	if err != nil {
		t.Fatalf("validate registered token: %v", err)
	}
	if principal == nil || principal.TokenID != created.Token.ID || principal.Username != "alice" {
		t.Fatalf("registered principal = %#v", principal)
	}

	unknown, err := auth.NewJWTService().GenerateOpenAPITokenWithContext(auth.UserTokenContext{
		UserID:   43,
		Username: "mallory",
		Email:    "mallory@example.com",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if principal, err := Validate(unknown, "203.0.113.10", "store-test"); err == nil {
		t.Fatalf("unregistered OpenAPI token was accepted: %#v", principal)
	}

	if err := Revoke("alice", created.Token.ID); err != nil {
		t.Fatal(err)
	}
	if principal, err := Validate(created.Secret, "203.0.113.10", "store-test"); err == nil {
		t.Fatalf("revoked OpenAPI token was accepted: %#v", principal)
	}
}

func useOpenAPITestDB(t *testing.T, database *gorm.DB) func() {
	t.Helper()
	dbMu.Lock()
	previous := db
	db = nil
	dbMu.Unlock()
	if err := SetDB(database); err != nil {
		t.Fatal(err)
	}
	return func() {
		dbMu.Lock()
		db = previous
		dbMu.Unlock()
	}
}
