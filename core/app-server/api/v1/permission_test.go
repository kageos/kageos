package v1

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/core/app-server/repository"
	"github.com/kageos/kageos/core/app-server/service"
	"github.com/kageos/kageos/pkg/access"
	"github.com/kageos/kageos/pkg/contextx"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestPermissionListAssignmentsAllowsReader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open("file:"+dbName+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.WorkspaceRoleAssignment{}); err != nil {
		t.Fatal(err)
	}
	assignment := &model.WorkspaceRoleAssignment{
		TenantUser:    "alice",
		App:           "ops",
		PrincipalType: string(access.PrincipalUser),
		PrincipalKey:  "bob",
		ResourcePath:  "/alice/ops",
		RoleCode:      string(access.RoleViewer),
	}
	assignment.CreatedBy = "alice"
	if err := db.Create(assignment).Error; err != nil {
		t.Fatal(err)
	}

	permissionService := service.NewPermissionService(
		repository.NewRoleAssignmentRepository(db),
		nil,
		nil,
	)
	handler := NewPermission(permissionService, nil)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(
		http.MethodGet,
		"/workspace/api/v1/permissions/assignments?resource_path=/alice/ops/tickets",
		nil,
	)
	ctx.Request.Header.Set(contextx.RequestUserHeader, "bob")
	ctx.Request.Header.Set(contextx.DepartmentFullPathHeader, "/org/unassigned")
	handler.ListAssignments(ctx)

	var got struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Assignments []access.RoleAssignmentView `json:"assignments"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, recorder.Body.String())
	}
	if got.Code != 0 {
		t.Fatalf("reader should list permission members: code=%d msg=%q", got.Code, got.Msg)
	}
	if len(got.Data.Assignments) != 1 || got.Data.Assignments[0].PrincipalKey != "bob" {
		t.Fatalf("assignments = %#v", got.Data.Assignments)
	}
}
