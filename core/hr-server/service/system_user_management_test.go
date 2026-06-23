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

func TestCreateUserFromSystemCreatesCompanyAndUser(t *testing.T) {
	t.Parallel()

	db := openSystemUserManagementTestDB(t)
	userRepo := repository.NewUserRepository(db)
	companyRepo := repository.NewCompanyRepository(db)
	svc := NewUserService(userRepo, companyRepo, nil, nil)

	user, err := svc.CreateUserFromSystem(context.Background(), dto.SystemCreateUserReq{
		Username:    "alice",
		Password:    "secret123",
		Nickname:    "Alice",
		CompanyCode: "acme",
		CompanyName: "Acme Inc",
	}, SystemUsername)
	if err != nil {
		t.Fatal(err)
	}
	if user.CompanyCode != "acme" {
		t.Fatalf("company code = %q, want acme", user.CompanyCode)
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

	company, err := companyRepo.GetCompanyByCode("acme")
	if err != nil {
		t.Fatal(err)
	}
	if company.Name != "Acme Inc" {
		t.Fatalf("company name = %q, want Acme Inc", company.Name)
	}
}

func TestUpdateUserStatusFromSystemDoesNotDisableSystem(t *testing.T) {
	t.Parallel()

	db := openSystemUserManagementTestDB(t)
	userRepo := repository.NewUserRepository(db)
	companyRepo := repository.NewCompanyRepository(db)
	svc := NewUserService(userRepo, companyRepo, nil, nil)
	if err := userRepo.CreateUser(&model.User{
		Username:      SystemUsername,
		Email:         SystemUserEmail,
		CompanyCode:   model.DefaultCompanyCode,
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

func openSystemUserManagementTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Company{}, &model.User{}); err != nil {
		t.Fatalf("migrate system user management test db: %v", err)
	}
	if err := db.Create(&model.Company{Code: model.DefaultCompanyCode, Name: "Default", CreatedBy: SystemUsername}).Error; err != nil {
		t.Fatalf("create default company: %v", err)
	}
	return db
}
