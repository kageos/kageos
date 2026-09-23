package repository

import (
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
	if err := repo.CreateUserSession(1, "expired-token", "expired-refresh", expiredAt, "", ""); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateUserSession(1, "active-token", "active-refresh", activeAt, "", ""); err != nil {
		t.Fatal(err)
	}

	if _, err := repo.GetUserSessionByToken("expired-token"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expired access token err = %v, want record not found", err)
	}
	if _, err := repo.GetUserSessionByRefreshToken("expired-refresh"); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expired refresh token err = %v, want record not found", err)
	}
	if _, err := repo.GetUserSessionByToken("active-token"); err != nil {
		t.Fatalf("active access token should be returned: %v", err)
	}
	if _, err := repo.GetUserSessionByRefreshToken("active-refresh"); err != nil {
		t.Fatalf("active refresh token should be returned: %v", err)
	}
}

func TestUserSessionRepositoryDeletesExpiredSessions(t *testing.T) {
	db := openUserSessionRepositoryTestDB(t)
	repo := NewUserSessionRepository(db)

	if err := repo.CreateUserSession(1, "expired-token", "expired-refresh", models.Time(time.Now().Add(-time.Hour)), "", ""); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateUserSession(1, "active-token", "active-refresh", models.Time(time.Now().Add(time.Hour)), "", ""); err != nil {
		t.Fatal(err)
	}

	if err := repo.DeleteExpiredSessions(); err != nil {
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

func TestLoginTimestampCommitsWithSession(t *testing.T) {
	db := openUserSessionRepositoryTestDB(t)
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatal(err)
	}
	user := model.User{Username: "login_user", Status: "active"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	repo := NewUserSessionRepository(db)
	read := func() *model.User {
		t.Helper()
		got, err := NewUserRepository(db).GetUserByID(user.ID)
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	if read().LastLoginAt != nil {
		t.Fatal("new user has a login timestamp")
	}
	start := time.Now().Truncate(time.Second)
	if err := repo.CreateActiveUserSession(user.ID, "login", "refresh", models.Time(time.Now().Add(time.Hour))); err != nil {
		t.Fatal(err)
	}
	loggedIn := read()
	if loggedIn.LastLoginAt == nil || time.Time(*loggedIn.LastLoginAt).Before(start) {
		t.Fatal("successful login was not recorded")
	}
	// Token refresh must preserve the actual login time.
	old := models.Time(time.Now().Add(-24 * time.Hour).Truncate(time.Second))
	if err := db.Model(&user).UpdateColumn("last_login_at", old).Error; err != nil {
		t.Fatal(err)
	}
	session, err := repo.GetUserSessionByToken("login")
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.UpdateUserSessionTokens(session.ID, "new-login", "new-refresh"); err != nil {
		t.Fatal(err)
	}
	if got := read(); got.LastLoginAt == nil || !time.Time(*got.LastLoginAt).Equal(time.Time(old)) {
		t.Fatal("refresh changed login time")
	}
	// A failed session insert must leave the previous timestamp intact.
	if err := repo.CreateActiveUserSession(user.ID, "new-login", "new-refresh", models.Time(time.Now().Add(time.Hour))); err == nil {
		t.Fatal("expected duplicate session failure")
	}
	if got := read(); got.LastLoginAt == nil || !time.Time(*got.LastLoginAt).Equal(time.Time(old)) {
		t.Fatal("failed login changed timestamp")
	}
	if err := repo.CreateActiveUserSession(user.ID, "second-login", "second-refresh", models.Time(time.Now().Add(time.Hour))); err != nil {
		t.Fatal(err)
	}
	if got := read(); got.LastLoginAt == nil || !time.Time(*got.LastLoginAt).After(time.Time(old)) {
		t.Fatal("second login did not advance timestamp")
	}
	if err := db.Model(&user).UpdateColumn("status", "disabled").Error; err != nil {
		t.Fatal(err)
	}
	before := read()
	if err := repo.CreateActiveUserSession(user.ID, "disabled-login", "disabled-refresh", models.Time(time.Now().Add(time.Hour))); err == nil {
		t.Fatal("disabled user logged in")
	}
	if got := read(); !time.Time(*got.LastLoginAt).Equal(time.Time(*before.LastLoginAt)) {
		t.Fatal("disabled login changed timestamp")
	}
}
