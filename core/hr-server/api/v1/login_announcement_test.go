package v1

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/core/hr-server/model"
	"github.com/kageos/kageos/core/hr-server/repository"
	"github.com/kageos/kageos/core/hr-server/service"
	"github.com/kageos/kageos/dto"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPublicLoginAnnouncementHidesDisabledDraft(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatal(err)
	}
	settingsService := service.NewSystemSettingsService(repository.NewSystemSettingRepository(db))
	if _, err := settingsService.UpdateLoginAnnouncement(dto.UpdateLoginAnnouncementReq{
		Markdown: "## 内部草稿\n\n不应公开",
	}, "system"); err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	NewLoginAnnouncement(settingsService).PublicGet(context)

	var body struct {
		Data dto.LoginAnnouncement `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Data != (dto.LoginAnnouncement{}) {
		t.Fatalf("disabled announcement leaked through public API: %#v", body.Data)
	}
}
