package service

import (
	"strings"
	"testing"

	"github.com/kageos/kageos/core/hr-server/model"
	"github.com/kageos/kageos/core/hr-server/repository"
	"github.com/kageos/kageos/dto"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newLoginAnnouncementService(t *testing.T) *SystemSettingsService {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatal(err)
	}
	return NewSystemSettingsService(repository.NewSystemSettingRepository(db))
}

func TestLoginAnnouncementDefaultsAndUpdate(t *testing.T) {
	service := newLoginAnnouncementService(t)

	initial, err := service.GetLoginAnnouncement()
	if err != nil {
		t.Fatal(err)
	}
	if *initial != (dto.LoginAnnouncement{}) {
		t.Fatalf("unexpected initial announcement: %#v", initial)
	}

	updated, err := service.UpdateLoginAnnouncement(dto.UpdateLoginAnnouncementReq{
		Enabled:  true,
		Markdown: "  ## 测试公告\n\n账号由现场人员提供。  ",
	}, "system")
	if err != nil {
		t.Fatal(err)
	}
	want := dto.LoginAnnouncement{Enabled: true, Markdown: "## 测试公告\n\n账号由现场人员提供。"}
	if *updated != want {
		t.Fatalf("unexpected updated announcement: got %#v, want %#v", updated, want)
	}
}

func TestLoginAnnouncementValidation(t *testing.T) {
	service := newLoginAnnouncementService(t)

	_, err := service.UpdateLoginAnnouncement(dto.UpdateLoginAnnouncementReq{Enabled: true}, "system")
	if err == nil || !strings.Contains(err.Error(), "markdown is required") {
		t.Fatalf("expected required markdown error, got %v", err)
	}

	_, err = service.UpdateLoginAnnouncement(dto.UpdateLoginAnnouncementReq{Markdown: strings.Repeat("文", 10001)}, "system")
	if err == nil || !strings.Contains(err.Error(), "markdown must not exceed") {
		t.Fatalf("expected markdown length error, got %v", err)
	}
}
