package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/kageos/kageos/core/hr-server/model"
	"github.com/kageos/kageos/core/hr-server/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestResolveExistingUserForPrincipalDoesNotAutoBindByEmail(t *testing.T) {
	t.Parallel()

	db := openOAuthServiceTestDB(t)
	userRepo := repository.NewUserRepository(db)
	identityRepo := repository.NewAuthExternalIdentityRepository(db)
	if err := userRepo.CreateUser(&model.User{
		Username:      "google_user",
		Email:         "same@example.com",
		Status:        "active",
		EmailVerified: true,
		RegisterType:  "google",
	}); err != nil {
		t.Fatal(err)
	}

	svc := &AuthOAuthService{
		identityRepo: identityRepo,
		userRepo:     userRepo,
	}
	user, found, err := svc.resolveExistingUserForPrincipal(ExternalPrincipal{
		ProviderCode:  ProviderGitHubOAuth,
		ExternalID:    "github-subject-1",
		Email:         "same@example.com",
		EmailVerified: true,
		Nickname:      "GitHub User",
	})
	if err != nil {
		t.Fatal(err)
	}
	if found || user != nil {
		t.Fatalf("resolveExistingUserForPrincipal found user by email, got found=%v user=%v", found, user)
	}

	var identityCount int64
	if err := db.Model(&model.AuthExternalIdentity{}).Count(&identityCount).Error; err != nil {
		t.Fatal(err)
	}
	if identityCount != 0 {
		t.Fatalf("created external identity by email match, count=%d", identityCount)
	}
}

func TestCreateRegistrationIntentAllowsMissingEmail(t *testing.T) {
	t.Parallel()

	db := openOAuthServiceTestDB(t)
	svc := &AuthOAuthService{
		registrationIntentRepo: repository.NewAuthOAuthRegistrationIntentRepository(db),
		userRepo:               repository.NewUserRepository(db),
	}

	intent, err := svc.createExternalRegistrationIntent(ExternalPrincipal{
		ProviderCode: ProviderGitHubOAuth,
		ExternalID:   "github-no-email",
		Nickname:     "No Email User",
	}, ExternalLoginOptions{
		ShortCode:     "github",
		RedirectAfter: "/workspace",
	})
	if err != nil {
		t.Fatal(err)
	}
	if intent.Email != "" {
		t.Fatalf("intent email = %q, want empty", intent.Email)
	}
	if intent.EmailVerified {
		t.Fatal("intent EmailVerified = true, want false for missing email")
	}
}

func TestCompleteExternalLoginCreatesRegistrationIntentForUnknownPrincipal(t *testing.T) {
	t.Parallel()

	db := openOAuthServiceTestDB(t)
	svc := &AuthOAuthService{
		registrationIntentRepo: repository.NewAuthOAuthRegistrationIntentRepository(db),
		identityRepo:           repository.NewAuthExternalIdentityRepository(db),
		userRepo:               repository.NewUserRepository(db),
	}

	result, err := svc.CompleteExternalLogin(context.Background(), ExternalPrincipal{
		ProviderCode: "custom_sso",
		ExternalID:   "subject-1",
		Username:     "alice",
		Email:        "Alice@Example.COM",
		Nickname:     "Alice",
	}, ExternalLoginOptions{
		ShortCode:     "sso",
		RedirectAfter: "/workspace",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.RegistrationRequired || result.RegistrationTicket == "" {
		t.Fatalf("registration result = %+v, want required with ticket", result)
	}

	intent, err := registrationIntentByTicket(db, result.RegistrationTicket)
	if err != nil {
		t.Fatal(err)
	}
	if intent.ProviderCode != "custom_sso" || intent.ExternalID != "subject-1" {
		t.Fatalf("intent identity = %s/%s", intent.ProviderCode, intent.ExternalID)
	}
	if intent.Email != "alice@example.com" {
		t.Fatalf("intent email = %q, want normalized email", intent.Email)
	}
}

func TestOAuthRegistrationCompleteAllowsDuplicateEmailContact(t *testing.T) {
	t.Parallel()

	db := openOAuthServiceTestDB(t)
	registrationRepo := repository.NewAuthOAuthRegistrationIntentRepository(db)

	for i := 1; i <= 2; i++ {
		ticket := fmt.Sprintf("ticket-%d", i)
		externalID := fmt.Sprintf("github-subject-%d", i)
		if err := registrationRepo.Create(&model.AuthOAuthRegistrationIntent{
			Ticket:        ticket,
			ProviderCode:  ProviderGitHubOAuth,
			ExternalID:    externalID,
			Email:         "same@example.com",
			EmailVerified: true,
			SuggestedCode: fmt.Sprintf("oauthuser%d", i),
			ExpiresAt:     time.Now().Add(time.Minute),
		}); err != nil {
			t.Fatal(err)
		}
		_, err := registrationRepo.Complete(ticket, &model.User{
			Username:      fmt.Sprintf("oauthuser%d", i),
			Email:         "same@example.com",
			Status:        "active",
			RegisterType:  "github",
			EmailVerified: true,
		}, &model.AuthExternalIdentity{
			ProviderCode: ProviderGitHubOAuth,
			ExternalID:   externalID,
			Email:        "same@example.com",
		})
		if err != nil {
			t.Fatalf("complete registration %d: %v", i, err)
		}
	}
}

func openOAuthServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{},
		&model.AuthExternalIdentity{},
		&model.AuthOAuthRegistrationIntent{},
	); err != nil {
		t.Fatalf("migrate oauth service test db: %v", err)
	}
	return db
}

func registrationIntentByTicket(db *gorm.DB, ticket string) (*model.AuthOAuthRegistrationIntent, error) {
	var intent model.AuthOAuthRegistrationIntent
	if err := db.Where("ticket = ?", ticket).First(&intent).Error; err != nil {
		return nil, err
	}
	return &intent, nil
}
