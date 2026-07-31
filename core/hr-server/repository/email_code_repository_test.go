package repository

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kageos/kageos/core/hr-server/model"
	"github.com/kageos/kageos/pkg/gormx/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestEmailCodeRejectsExpiredCode(t *testing.T) {
	repo, _ := openEmailCodeRepositoryTestDB(t)
	if err := repo.CreateEmailCode(
		"alice@example.com",
		"123456",
		models.Time(time.Now().Add(-time.Minute)),
		"register",
		"127.0.0.1",
		"test",
	); err != nil {
		t.Fatal(err)
	}

	err := repo.VerifyAndConsumeLatestEmailCode("alice@example.com", "123456", "register", 5)
	if !errors.Is(err, ErrEmailCodeInvalid) {
		t.Fatalf("expired code error = %v, want ErrEmailCodeInvalid", err)
	}
}

func TestEmailCodeCanOnlyBeConsumedOnceConcurrently(t *testing.T) {
	repo, _ := openEmailCodeRepositoryTestDB(t)
	if err := repo.CreateEmailCode(
		"alice@example.com",
		"123456",
		models.Time(time.Now().Add(time.Minute)),
		"register",
		"127.0.0.1",
		"test",
	); err != nil {
		t.Fatal(err)
	}

	var successes atomic.Int32
	var wait sync.WaitGroup
	for i := 0; i < 2; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			if repo.VerifyAndConsumeLatestEmailCode("alice@example.com", "123456", "register", 5) == nil {
				successes.Add(1)
			}
		}()
	}
	wait.Wait()
	if got := successes.Load(); got != 1 {
		t.Fatalf("successful consumes = %d, want exactly 1", got)
	}
}

func TestEmailCodeStopsAfterFailedAttemptLimit(t *testing.T) {
	repo, db := openEmailCodeRepositoryTestDB(t)
	if err := repo.CreateEmailCode(
		"alice@example.com",
		"123456",
		models.Time(time.Now().Add(time.Minute)),
		"register",
		"127.0.0.1",
		"test",
	); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		if err := repo.VerifyAndConsumeLatestEmailCode("alice@example.com", "000000", "register", 5); err == nil {
			t.Fatal("wrong code unexpectedly succeeded")
		}
	}
	if err := repo.VerifyAndConsumeLatestEmailCode("alice@example.com", "123456", "register", 5); !errors.Is(err, ErrEmailCodeTooManyTries) {
		t.Fatalf("attempt-limited code error = %v, want ErrEmailCodeTooManyTries", err)
	}
	var emailCode model.EmailCode
	if err := db.First(&emailCode).Error; err != nil {
		t.Fatal(err)
	}
	if emailCode.Attempts != 5 || emailCode.Used {
		t.Fatalf("email code state = attempts:%d used:%v", emailCode.Attempts, emailCode.Used)
	}
}

func TestDeleteExpiredEmailCodesUsesCurrentTime(t *testing.T) {
	repo, db := openEmailCodeRepositoryTestDB(t)
	for code, expiresAt := range map[string]time.Time{
		"111111": time.Now().Add(-time.Minute),
		"222222": time.Now().Add(time.Minute),
	} {
		if err := repo.CreateEmailCode("alice@example.com", code, models.Time(expiresAt), "register", "", ""); err != nil {
			t.Fatal(err)
		}
	}
	if err := repo.DeleteExpiredEmailCodes(); err != nil {
		t.Fatal(err)
	}
	var codes []model.EmailCode
	if err := db.Find(&codes).Error; err != nil {
		t.Fatal(err)
	}
	if len(codes) != 1 || codes[0].Code != "222222" {
		t.Fatalf("remaining codes = %+v, want only active code", codes)
	}
}

func TestInvalidateEmailCodeKeepsRateLimitRecordButRestoresPreviousCode(t *testing.T) {
	repo, db := openEmailCodeRepositoryTestDB(t)
	expiresAt := models.Time(time.Now().Add(time.Minute))
	if err := repo.CreateEmailCode("alice@example.com", "111111", expiresAt, "register", "", ""); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateEmailCode("alice@example.com", "222222", expiresAt, "register", "", ""); err != nil {
		t.Fatal(err)
	}
	if err := repo.InvalidateEmailCode("alice@example.com", "222222", "register"); err != nil {
		t.Fatal(err)
	}
	if err := repo.VerifyAndConsumeLatestEmailCode("alice@example.com", "111111", "register", 5); err != nil {
		t.Fatalf("previous delivered code should remain usable: %v", err)
	}

	var count int64
	if err := db.Model(&model.EmailCode{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("email code rows = %d, want both attempts retained for rate limiting", count)
	}
}

func openEmailCodeRepositoryTestDB(t *testing.T) (*EmailCodeRepository, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(&model.EmailCode{}); err != nil {
		t.Fatalf("migrate email_code: %v", err)
	}
	return NewEmailCodeRepository(db), db
}
