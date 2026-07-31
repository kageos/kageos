package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/core/app-server/repository"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/access"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newPermissionRequestTestService(t *testing.T) (*PermissionRequestService, *PermissionService, *gorm.DB) {
	t.Helper()
	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open("file:"+dbName+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := db.AutoMigrate(
		&model.App{},
		&model.ServiceTree{},
		&model.WorkspaceRoleAssignment{},
		&model.WorkspacePermissionRequest{},
		&model.OperateLog{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	appRepo := repository.NewAppRepository(db)
	appModel := &model.App{User: "alice", Code: "ops", Name: "Ops", Version: "v1", Admins: "bob"}
	if err := appRepo.CreateApp(appModel); err != nil {
		t.Fatalf("create app: %v", err)
	}
	resource := &model.ServiceTree{
		Name:         "工单提交",
		Code:         "submit.form",
		Type:         model.ServiceTreeTypeFunction,
		AppID:        appModel.ID,
		RefID:        1,
		FullCodePath: "/alice/ops/tickets/submit.form",
	}
	if err := db.Create(resource).Error; err != nil {
		t.Fatalf("create service tree: %v", err)
	}

	roleRepo := repository.NewRoleAssignmentRepository(db)
	permission := NewPermissionService(roleRepo, repository.NewOperateLogRepository(db), appRepo)
	permission.userLookup = func(ctx context.Context, username string) (*dto.UserInfo, error) {
		return &dto.UserInfo{Username: username}, nil
	}
	requestService := NewPermissionRequestService(
		repository.NewPermissionRequestRepository(db),
		roleRepo,
		repository.NewServiceTreeRepository(db),
		appRepo,
		permission,
	)
	return requestService, permission, db
}

func TestPermissionRequestApproveGrantsRoleAndRecordsReviewer(t *testing.T) {
	requestService, permission, db := newPermissionRequestTestService(t)
	resourcePath := "/alice/ops/tickets/submit.form"

	created, err := requestService.CreateRequest(
		actorContext("carol"),
		"alice",
		"ops",
		"carol",
		resourcePath,
		access.RoleViewer,
		"需要提交工单",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if created.Status != model.PermissionRequestStatusPending {
		t.Fatalf("status = %q", created.Status)
	}
	if len(created.Approvers) < 2 {
		t.Fatalf("approvers = %#v, want workspace owner and admin", created.Approvers)
	}

	pending, err := requestService.ListPendingForReviewer(actorContext("bob"), "alice", "ops", "bob")
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 || pending[0].ID != created.ID {
		t.Fatalf("pending = %#v", pending)
	}

	approved, err := requestService.Approve(actorContext("bob"), created.ID, "bob", "同意使用")
	if err != nil {
		t.Fatal(err)
	}
	if approved.Status != model.PermissionRequestStatusApproved || approved.ReviewedBy != "bob" {
		t.Fatalf("approved = %#v", approved)
	}

	canRead, err := permission.HasPermission(context.Background(), "alice", "ops", "carol", resourcePath, access.ActionRead)
	if err != nil {
		t.Fatal(err)
	}
	if !canRead {
		t.Fatal("approved request should grant read permission")
	}

	var assignment model.WorkspaceRoleAssignment
	if err := db.Where("principal_key = ? AND resource_path = ?", "carol", resourcePath).First(&assignment).Error; err != nil {
		t.Fatal(err)
	}
	if assignment.CreatedBy != "bob" {
		t.Fatalf("assignment created_by = %q", assignment.CreatedBy)
	}
}

func TestPermissionRequestRejectAndCancelDoNotGrantRole(t *testing.T) {
	requestService, permission, _ := newPermissionRequestTestService(t)
	resourcePath := "/alice/ops/tickets/submit.form"

	rejectedRequest, err := requestService.CreateRequest(
		actorContext("carol"), "alice", "ops", "carol", resourcePath,
		access.RoleViewer, "第一次申请", nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := requestService.Reject(actorContext("bob"), rejectedRequest.ID, "bob", "用途不明确"); err != nil {
		t.Fatal(err)
	}

	cancelledRequest, err := requestService.CreateRequest(
		actorContext("carol"), "alice", "ops", "carol", resourcePath,
		access.RoleViewer, "重新申请", nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := requestService.Cancel(actorContext("carol"), cancelledRequest.ID, "carol"); err != nil {
		t.Fatal(err)
	}

	canRead, err := permission.HasPermission(context.Background(), "alice", "ops", "carol", resourcePath, access.ActionRead)
	if err != nil {
		t.Fatal(err)
	}
	if canRead {
		t.Fatal("rejected and cancelled requests must not grant permission")
	}

	mine, err := requestService.ListMine(actorContext("carol"), "alice", "ops", "carol", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(mine) != 2 || mine[0].Status != model.PermissionRequestStatusCancelled || mine[1].Status != model.PermissionRequestStatusRejected {
		t.Fatalf("mine = %#v", mine)
	}
}

func TestPermissionRequestRejectsDuplicateAndUnauthorizedReviewer(t *testing.T) {
	requestService, _, _ := newPermissionRequestTestService(t)
	resourcePath := "/alice/ops/tickets/submit.form"

	created, err := requestService.CreateRequest(
		actorContext("carol"), "alice", "ops", "carol", resourcePath,
		access.RoleMember, "需要维护工单", nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := requestService.CreateRequest(
		actorContext("carol"), "alice", "ops", "carol", resourcePath,
		access.RoleMember, "重复申请", nil,
	); err == nil || !strings.Contains(err.Error(), "已有待审批申请") {
		t.Fatalf("duplicate error = %v", err)
	}

	if _, err := requestService.Approve(actorContext("dave"), created.ID, "dave", ""); err == nil || !strings.Contains(err.Error(), "不是该资源的审批人") {
		t.Fatalf("unauthorized review error = %v", err)
	}
	if _, err := requestService.Approve(actorContext("carol"), created.ID, "carol", ""); err == nil || !strings.Contains(err.Error(), "不能审批自己的权限申请") {
		t.Fatalf("self review error = %v", err)
	}

	pending, err := requestService.ListPendingForReviewer(actorContext("dave"), "alice", "ops", "dave")
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 0 {
		t.Fatalf("unauthorized reviewer pending = %#v", pending)
	}
}

func TestPermissionRequestSupportsWorkspaceRootAndRefusesExpiredApproval(t *testing.T) {
	requestService, permission, db := newPermissionRequestTestService(t)
	resourcePath := "/alice/ops"
	expiresAt := time.Now().Add(time.Hour)

	created, err := requestService.CreateRequest(
		actorContext("carol"), "alice", "ops", "carol", resourcePath,
		access.RoleViewer, "需要进入工作空间", &expiresAt,
	)
	if err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-time.Minute)
	if err := db.Model(&model.WorkspacePermissionRequest{}).
		Where("id = ?", created.ID).
		Update("requested_expires", past).Error; err != nil {
		t.Fatal(err)
	}

	if _, err := requestService.Approve(actorContext("bob"), created.ID, "bob", ""); err == nil || !strings.Contains(err.Error(), "有效期已过") {
		t.Fatalf("expired approval error = %v", err)
	}
	canRead, err := permission.HasPermission(context.Background(), "alice", "ops", "carol", resourcePath, access.ActionRead)
	if err != nil {
		t.Fatal(err)
	}
	if canRead {
		t.Fatal("expired approval must not grant permission")
	}
}
