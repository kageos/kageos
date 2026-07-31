package openapitoken

import (
	"errors"
	"strings"
	"testing"

	"github.com/kageos/kageos/pkg/auth"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestNewStoreRejectsNilDatabase(t *testing.T) {
	if _, err := NewStore(nil); err == nil {
		t.Fatal("NewStore(nil) should fail")
	}
}

func TestValidateRequiresExistingActiveRecord(t *testing.T) {
	store := newTestStore(t, "active")
	created, err := store.Create(CreateInput{
		OwnerUserID:   42,
		OwnerUsername: "alice",
		Name:          "automation",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(created.Token.TokenPrefix, TokenPrefix) {
		t.Fatalf("display prefix = %q, want %s fingerprint", created.Token.TokenPrefix, TokenPrefix)
	}

	principal, err := store.Validate(created.Secret, "127.0.0.1", "test")
	if err != nil {
		t.Fatalf("active token should validate: %v", err)
	}
	if principal.TokenID != created.Token.ID || principal.UserID != 42 || principal.Username != "alice" {
		t.Fatalf("principal = %#v", principal)
	}

	emptyStore := newTestStore(t, "empty")
	if _, err := emptyStore.Validate(created.Secret, "", ""); !errors.Is(err, ErrTokenNotFound) {
		t.Fatalf("missing record error = %v, want ErrTokenNotFound", err)
	}
}

func TestValidateRejectsRevokedRecord(t *testing.T) {
	store := newTestStore(t, "revoked")
	created, err := store.Create(CreateInput{
		OwnerUserID:   42,
		OwnerUsername: "alice",
		Name:          "automation",
	})
	if err != nil {
		t.Fatal(err)
	}
	revoked, err := store.RevokeWithResult("alice", created.Token.ID)
	if err != nil {
		t.Fatal(err)
	}
	if revoked.TokenHash != HashToken(created.Secret) {
		t.Fatalf("revoked hash = %q, want created token hash", revoked.TokenHash)
	}
	if _, err := store.Validate(created.Secret, "", ""); !errors.Is(err, ErrTokenRevoked) {
		t.Fatalf("revoked token error = %v, want ErrTokenRevoked", err)
	}
}

func TestValidateRejectsNonOpenAPIToken(t *testing.T) {
	store := newTestStore(t, "wrong_type")
	accessToken, err := auth.NewJWTService().GenerateAccessToken(42, "alice", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Validate(accessToken, "", ""); err == nil {
		t.Fatal("access token must not validate as an OpenAPI Token")
	}
}

func TestStoresKeepDatabaseStateIsolated(t *testing.T) {
	storeA := newTestStore(t, "a")
	storeB := newTestStore(t, "b")

	record := &OpenAPIToken{
		OwnerUsername: "alice",
		Name:          "automation",
		TokenPrefix:   "kgos_test_a",
		TokenHash:     strings.Repeat("a", 64),
	}
	if err := storeA.db.Create(record).Error; err != nil {
		t.Fatalf("create token: %v", err)
	}

	tokensA, err := storeA.List("alice")
	if err != nil {
		t.Fatalf("list store A: %v", err)
	}
	tokensB, err := storeB.List("alice")
	if err != nil {
		t.Fatalf("list store B: %v", err)
	}
	if len(tokensA) != 1 {
		t.Fatalf("store A token count = %d, want 1", len(tokensA))
	}
	if len(tokensB) != 0 {
		t.Fatalf("store B token count = %d, want 0", len(tokensB))
	}
}

func newTestStore(t *testing.T, suffix string) *Store {
	t.Helper()
	databaseName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name()) + "_" + suffix
	db, err := gorm.Open(sqlite.Open("file:"+databaseName+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	store, err := NewStore(db)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	return store
}
