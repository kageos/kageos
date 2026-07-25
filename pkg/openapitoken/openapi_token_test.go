package openapitoken

import (
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestNewStoreRejectsNilDatabase(t *testing.T) {
	if _, err := NewStore(nil); err == nil {
		t.Fatal("NewStore(nil) should fail")
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
