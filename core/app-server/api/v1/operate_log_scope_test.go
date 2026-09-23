package v1

import (
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/core/app-server/repository"
	"github.com/kageos/kageos/core/app-server/service"
	"github.com/kageos/kageos/pkg/contextx"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestGlobalOperateLogScopeRequiresSystemAndSurvivesDeletedWorkspace(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "logs.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.OperateLog{}); err != nil {
		t.Fatal(err)
	}
	// No app or service_tree tables: the resource no longer exists.
	if err := db.Create(&model.OperateLog{TenantUser: "alice", App: "deleted", ActorUser: "alice", Action: "workspace.deleted", ResourcePath: "/alice/deleted", Status: "success"}).Error; err != nil {
		t.Fatal(err)
	}
	handler := NewOperateLog(service.NewOperateLogService(repository.NewOperateLogRepository(db)), nil)
	for _, actor := range []string{"alice", "system"} {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/logs?resource_path_prefix=%2F", nil)
		c.Set(contextx.RequestUserHeader, actor)
		handler.GetOperateLogs(c)
		var result struct {
			Code int `json:"code"`
			Data struct {
				Total int `json:"total"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if actor == "system" && (result.Code != 0 || result.Data.Total != 1) {
			t.Fatalf("system cannot retrieve deleted history: %s", w.Body.String())
		}
		if actor != "system" && result.Code == 0 {
			t.Fatal("non-admin gained global access")
		}
	}
}

func TestArchiveDownloadRejectsNonAdminBeforeResolvingFile(t *testing.T) {
	handler := NewLogArchive(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/system/log_archives/1/download", nil)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Set(contextx.RequestUserHeader, "alice")
	handler.Download(c)
	var result struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Code == 0 {
		t.Fatalf("unauthorized download allowed: %s", w.Body.String())
	}
}
