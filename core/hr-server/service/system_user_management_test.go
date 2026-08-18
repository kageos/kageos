package service

import (
	"context"
	"strings"
	"testing"

	"github.com/kageos/kageos/core/hr-server/model"
	"github.com/kageos/kageos/core/hr-server/repository"
	"github.com/kageos/kageos/dto"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCreateUserFromSystemCreatesUser(t *testing.T) {
	t.Parallel()

	db := openSystemUserManagementTestDB(t)
	userRepo := repository.NewUserRepository(db)
	svc := NewUserService(userRepo, nil, nil, nil)

	user, err := svc.CreateUserFromSystem(context.Background(), dto.SystemCreateUserReq{
		Username: "alice",
		Password: "secret123",
		Nickname: "Alice",
	}, SystemUsername)
	if err != nil {
		t.Fatal(err)
	}
	if user.Status != "active" || !user.EmailVerified {
		t.Fatalf("status/email_verified = %q/%v, want active/true", user.Status, user.EmailVerified)
	}
	if user.DepartmentFullPath != "/org/unassigned" {
		t.Fatalf("department = %q, want /org/unassigned", user.DepartmentFullPath)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("secret123")); err != nil {
		t.Fatalf("password hash mismatch: %v", err)
	}
}

func TestUpdateUserStatusFromSystemDoesNotDisableSystem(t *testing.T) {
	t.Parallel()

	db := openSystemUserManagementTestDB(t)
	userRepo := repository.NewUserRepository(db)
	svc := NewUserService(userRepo, nil, nil, nil)
	if err := userRepo.CreateUser(&model.User{
		Username:      SystemUsername,
		Email:         SystemUserEmail,
		Status:        "active",
		RegisterType:  "system",
		EmailVerified: true,
		Type:          model.UserTypeSystem,
	}); err != nil {
		t.Fatal(err)
	}

	_, err := svc.UpdateUserStatusFromSystem(context.Background(), SystemUsername, "disabled")
	if err == nil || !strings.Contains(err.Error(), "不能停用 system 用户") {
		t.Fatalf("expected system disable error, got %v", err)
	}
}

func TestChangeOwnPassword(t *testing.T) {
	t.Parallel()

	db := openSystemUserManagementTestDB(t)
	userRepo := repository.NewUserRepository(db)
	sessionRepo := repository.NewUserSessionRepository(db)
	svc := NewUserService(userRepo, nil, sessionRepo, nil)
	oldHash, err := bcrypt.GenerateFromPassword([]byte("old-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	user := &model.User{
		Username:      "alice",
		Email:         "alice@example.com",
		PasswordHash:  string(oldHash),
		Status:        "active",
		RegisterType:  "system",
		EmailVerified: true,
		Type:          model.UserTypeNormal,
	}
	if err := userRepo.CreateUser(user); err != nil {
		t.Fatal(err)
	}

	if err := svc.ChangeOwnPassword(context.Background(), user.Username, "wrong-password", "new-password"); err == nil || !strings.Contains(err.Error(), "当前密码错误") {
		t.Fatalf("expected current password error, got %v", err)
	}
	if err := svc.ChangeOwnPassword(context.Background(), user.Username, "old-password", "new-password"); err != nil {
		t.Fatal(err)
	}
	updated, err := userRepo.GetUserByUsername(user.Username)
	if err != nil {
		t.Fatal(err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(updated.PasswordHash), []byte("new-password")); err != nil {
		t.Fatalf("new password hash mismatch: %v", err)
	}
}

func openSystemUserManagementTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}, &model.UserSession{}); err != nil {
		t.Fatalf("migrate system user management test db: %v", err)
	}
	return db
}
