package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kageos/kageos/core/hr-server/model"
	"github.com/kageos/kageos/pkg/gormx/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUserSessionRepositoryFiltersExpiredSessions(t *testing.T) {
	db := openUserSessionRepositoryTestDB(t)
	repo := NewUserSessionRepository(db)

	expiredAt := models.Time(time.Now().Add(-time.Hour))
	activeAt := models.Time(time.Now().Add(time.Hour))
	if err := repo.CreateUserSession(context.Background(), 1, "expired-token", "expired-refresh", expiredAt, "", ""); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateUserSession(context.Background(), 1, "active-token", "active-refresh", activeAt, "", ""); err != nil {
		t.Fatal(err)
	}

	if _, err := repo.GetUserSessionByToken(context.Background(), "expired-token"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expired access token err = %v, want record not found", err)
	}
	if _, err := repo.GetUserSessionByRefreshToken(context.Background(), "expired-refresh"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expired refresh token err = %v, want record not found", err)
	}
	if _, err := repo.GetUserSessionByToken(context.Background(), "active-token"); err != nil {
		t.Fatalf("active access token should be returned: %v", err)
	}
	if _, err := repo.GetUserSessionByRefreshToken(context.Background(), "active-refresh"); err != nil {
		t.Fatalf("active refresh token should be returned: %v", err)
	}
}

func TestUserSessionRepositoryDeletesExpiredSessions(t *testing.T) {
	db := openUserSessionRepositoryTestDB(t)
	repo := NewUserSessionRepository(db)

	if err := repo.CreateUserSession(context.Background(), 1, "expired-token", "expired-refresh", models.Time(time.Now().Add(-time.Hour)), "", ""); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateUserSession(context.Background(), 1, "active-token", "active-refresh", models.Time(time.Now().Add(time.Hour)), "", ""); err != nil {
		t.Fatal(err)
	}

	if err := repo.DeleteExpiredSessions(context.Background()); err != nil {
		t.Fatal(err)
	}

	var count int64
	if err := db.Model(&model.UserSession{}).Where("token = ?", "expired-token").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expired sessions remaining = %d, want 0", count)
	}
	if err := db.Model(&model.UserSession{}).Where("token = ?", "active-token").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("active sessions remaining = %d, want 1", count)
	}
}

func openUserSessionRepositoryTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.UserSession{}); err != nil {
		t.Fatalf("migrate user_session: %v", err)
	}
	return db
}
