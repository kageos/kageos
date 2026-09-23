package v1

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/core/app-server/repository"
	"github.com/kageos/kageos/core/app-server/service"
	"github.com/kageos/kageos/pkg/contextx"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestResourceArchivesRequireAdminAndStayInsideScope(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "archives.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.LogArchiveBatch{}, &model.WorkspaceRoleAssignment{}); err != nil {
		t.Fatal(err)
	}
	for _, grant := range []struct{ user, path, role string }{{"bob", "/alice/ops/team_x", "admin"}, {"carol", "/alice/ops/team_x", "viewer"}, {"dave", "/alice/ops/team_x/a.form", "admin"}} {
		row := model.WorkspaceRoleAssignment{TenantUser: "alice", App: "ops", PrincipalType: "user", PrincipalKey: grant.user, ResourcePath: grant.path, RoleCode: grant.role}
		if err := db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	for i, path := range []string{"/alice/ops/team_x/a.form", "/alice/ops/team_x/b.form", "/alice/ops/teamax/c.form", "/alice/ops/team_y/d.form", ""} {
		row := model.LogArchiveBatch{ArchiveKey: string(rune('a' + i)), TenantUser: "alice", App: "ops", ResourcePath: path, SelectedIDsJSON: json.RawMessage(`[]`), Status: "completed"}
		if err := db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	permission := service.NewPermissionService(repository.NewRoleAssignmentRepository(db), nil, nil)
	handler := NewLogArchive(service.NewLogArchiveService(repository.NewLogArchiveRepository(db), service.DefaultLogArchiveConfig()), permission)
	ctx := func(user, url string) (*gin.Context, *httptest.ResponseRecorder) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", url, nil)
		c.Set(contextx.RequestUserHeader, user)
		c.Request.Header.Set(contextx.RequestUserHeader, user)
		c.Request.Header.Set(contextx.DepartmentFullPathHeader, "/org/unassigned")
		return c, w
	}
	for _, item := range []struct {
		user, path string
		total      int
		allow      bool
	}{{"bob", "/alice/ops/team_x", 2, true}, {"dave", "/alice/ops/team_x/a.form", 1, true}, {"carol", "/alice/ops/team_x", 0, false}, {"bob", "/alice/ops", 0, false}} {
		c, w := ctx(item.user, "/log_archives?resource_path="+item.path)
		handler.List(c)
		var got struct {
			Code int `json:"code"`
			Data struct {
				Total int `json:"total"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if (got.Code == 0) != item.allow || item.allow && got.Data.Total != item.total {
			t.Fatalf("%s %s: %s", item.user, item.path, w.Body.String())
		}
	}
	for _, item := range []struct {
		user  string
		id    int64
		allow bool
	}{{"bob", 1, true}, {"bob", 3, false}, {"bob", 4, false}, {"bob", 5, false}, {"dave", 2, false}, {"carol", 1, false}, {"alice", 5, true}} {
		c, _ := ctx(item.user, "/log_archives/download")
		if err := handler.authorizeBatch(c, item.id); (err == nil) != item.allow {
			t.Fatalf("batch permissions %s/%d: %v", item.user, item.id, err)
		}
	}
	c, _ := ctx("bob", "/log_archives/download?resource_path=/alice/ops/team_x/b.form")
	if err := handler.authorizeBatch(c, 1); err == nil {
		t.Fatal("batch escaped requested scope")
	}
}
