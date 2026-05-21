package model

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestNatsURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		nats Nats
		want string
	}{
		{
			name: "without auth",
			nats: Nats{Host: "127.0.0.1", Port: 4222},
			want: "nats://127.0.0.1:4222",
		},
		{
			name: "with auth",
			nats: Nats{Host: "nats.internal", Port: 4222, User: "aos", Password: "p@ss word"},
			want: "nats://aos:p%40ss%20word@nats.internal:4222",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.nats.URL(); got != tt.want {
				t.Fatalf("URL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReconcileNatsHostFromURL(t *testing.T) {
	t.Parallel()

	db := openNatsTestDB(t)
	if err := db.Create(&Nats{Host: "localhost", Port: 4222}).Error; err != nil {
		t.Fatalf("create nats: %v", err)
	}

	if err := ReconcileNatsHostFromURL(db, "nats://kageos:p%40ss%20word@127.0.0.1:4222"); err != nil {
		t.Fatalf("ReconcileNatsHostFromURL() error = %v", err)
	}

	var got Nats
	if err := db.First(&got).Error; err != nil {
		t.Fatalf("query nats: %v", err)
	}
	if got.Host != "127.0.0.1" || got.Port != 4222 || got.User != "kageos" || got.Password != "p@ss word" {
		t.Fatalf("reconciled nats = %+v", got)
	}
}

func TestReconcileNatsHostFromURLEmptyNoop(t *testing.T) {
	t.Parallel()

	db := openNatsTestDB(t)
	if err := db.Create(&Nats{Host: "localhost", Port: 4222, User: "old", Password: "old-pass"}).Error; err != nil {
		t.Fatalf("create nats: %v", err)
	}

	if err := ReconcileNatsHostFromURL(db, ""); err != nil {
		t.Fatalf("ReconcileNatsHostFromURL() error = %v", err)
	}

	var got Nats
	if err := db.First(&got).Error; err != nil {
		t.Fatalf("query nats: %v", err)
	}
	if got.User != "old" || got.Password != "old-pass" {
		t.Fatalf("empty URL should not update credentials, got %+v", got)
	}
}

func openNatsTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Nats{}); err != nil {
		t.Fatalf("migrate nats: %v", err)
	}
	return db
}
