package service

import (
	"context"
	"testing"

	hrmodel "github.com/kageos/kageos/core/hr-server/model"
	hrrepository "github.com/kageos/kageos/core/hr-server/repository"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestInitSystemUserWithPasswordUpdatesExistingMismatchedPassword(t *testing.T) {
	t.Parallel()

	db := openSystemUserTestDB(t)
	oldHash, err := bcrypt.GenerateFromPassword([]byte("old-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	userRepo := hrrepository.NewUserRepository(db)
	if err := userRepo.CreateUser(context.Background(), &hrmodel.User{
		Username:     SystemUsername,
		Email:        SystemUserEmail,
		CompanyCode:  hrmodel.DefaultCompanyCode,
		PasswordHash: string(oldHash),
		Status:       "active",
		RegisterType: "system",
		Type:         hrmodel.UserTypeNormal,
	}); err != nil {
		t.Fatal(err)
	}

	if err := initSystemUserWithPassword(context.Background(), db, "new-password", false); err != nil {
		t.Fatal(err)
	}

	got, err := userRepo.GetUserByUsername(context.Background(), SystemUsername)
	if err != nil {
		t.Fatal(err)
	}
	if got.Type != hrmodel.UserTypeSystem {
		t.Fatalf("system user type = %v, want %v", got.Type, hrmodel.UserTypeSystem)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(got.PasswordHash), []byte("new-password")); err != nil {
		t.Fatalf("system password was not updated: %v", err)
	}
}

func TestInitTestUserWithPasswordUpdatesExistingMismatchedPassword(t *testing.T) {
	t.Parallel()

	db := openSystemUserTestDB(t)
	oldHash, err := bcrypt.GenerateFromPassword([]byte("old-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	userRepo := hrrepository.NewUserRepository(db)
	if err := userRepo.CreateUser(context.Background(), &hrmodel.User{
		Username:           TestUsername,
		Email:              TestUserEmail,
		CompanyCode:        hrmodel.DefaultCompanyCode,
		PasswordHash:       string(oldHash),
		Status:             "active",
		RegisterType:       "system",
		Type:               hrmodel.UserTypeNormal,
		DepartmentFullPath: "/old",
	}); err != nil {
		t.Fatal(err)
	}

	if err := initTestUserWithPassword(context.Background(), db, "new-password", false); err != nil {
		t.Fatal(err)
	}

	got, err := userRepo.GetUserByUsername(context.Background(), TestUsername)
	if err != nil {
		t.Fatal(err)
	}
	if got.DepartmentFullPath != TestUserDepartmentPath {
		t.Fatalf("test_user department = %q, want %q", got.DepartmentFullPath, TestUserDepartmentPath)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(got.PasswordHash), []byte("new-password")); err != nil {
		t.Fatalf("test_user password was not updated: %v", err)
	}
}

func TestInitDefaultCompanyUsesConfiguredValues(t *testing.T) {
	t.Setenv("KAGEOS_COMPANY_CODE", "acme")
	t.Setenv("KAGEOS_COMPANY_NAME", "Acme Inc")

	db := openSystemUserTestDB(t)
	if err := db.AutoMigrate(&hrmodel.Company{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&hrmodel.Company{Code: hrmodel.DefaultCompanyCode, Name: "Default", CreatedBy: "system"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&hrmodel.User{Username: "alice", Email: "alice@example.com", CompanyCode: hrmodel.DefaultCompanyCode, Status: "active"}).Error; err != nil {
		t.Fatal(err)
	}

	if err := InitDefaultCompany(context.Background(), db); err != nil {
		t.Fatal(err)
	}

	var company hrmodel.Company
	if err := db.Where("code = ?", "acme").First(&company).Error; err != nil {
		t.Fatal(err)
	}
	if company.Name != "Acme Inc" {
		t.Fatalf("company name = %q", company.Name)
	}
	var user hrmodel.User
	if err := db.Where("username = ?", "alice").First(&user).Error; err != nil {
		t.Fatal(err)
	}
	if user.CompanyCode != "acme" {
		t.Fatalf("user company code = %q", user.CompanyCode)
	}
}

func openSystemUserTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&hrmodel.User{}); err != nil {
		t.Fatalf("migrate user: %v", err)
	}
	return db
}
