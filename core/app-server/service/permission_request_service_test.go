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
	notifier := &recordingPermissionNotifier{}
	permission.permissionNotifier = notifier
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
	if len(notifier.notifications) != 1 {
		t.Fatalf("notifications = %#v", notifier.notifications)
	}
	if notifier.notifications[0].ToUser != "carol" {
		t.Fatalf("notification recipient = %q", notifier.notifications[0].ToUser)
	}
	requireNotificationContains(t, notifier.notifications[0], "权限申请已通过", resourcePath, "查看者", "bob", "同意使用")

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

func TestPermissionRequestWorkspaceSummaryGroupsMineAndReviewableCounts(t *testing.T) {
	requestService, _, _ := newPermissionRequestTestService(t)
	resourcePath := "/alice/ops/tickets/submit.form"
	for _, requester := range []string{"carol", "dave"} {
		if _, err := requestService.CreateRequest(
			actorContext(requester),
			"alice",
			"ops",
			requester,
			resourcePath,
			access.RoleViewer,
			"需要查看工单",
			nil,
		); err != nil {
			t.Fatal(err)
		}
	}

	carolSummary, err := requestService.WorkspaceSummary(
		actorContext("carol"), "alice", "ops", "carol",
	)
	if err != nil {
		t.Fatal(err)
	}
	if carolSummary.OwnPendingCount != 1 || carolSummary.ReviewPendingCount != 0 {
		t.Fatalf("carol summary = %#v", carolSummary)
	}
	if got := carolSummary.Paths[resourcePath]; got.OwnPendingCount != 1 || got.ReviewPendingCount != 0 {
		t.Fatalf("carol path summary = %#v", got)
	}

	bobSummary, err := requestService.WorkspaceSummary(
		actorContext("bob"), "alice", "ops", "bob",
	)
	if err != nil {
		t.Fatal(err)
	}
	if bobSummary.OwnPendingCount != 0 || bobSummary.ReviewPendingCount != 2 {
		t.Fatalf("bob summary = %#v", bobSummary)
	}
	if got := bobSummary.Paths[resourcePath]; got.OwnPendingCount != 0 || got.ReviewPendingCount != 2 {
		t.Fatalf("bob path summary = %#v", got)
	}
}

func TestPermissionRequestApproveSystemDirectoryMemberGrantsChildFormWrite(t *testing.T) {
	requestService, permission, db := newPermissionRequestTestService(t)
	appModel := &model.App{User: "system", Code: "democase", Name: "Demo", Version: "v1"}
	if err := db.Create(appModel).Error; err != nil {
		t.Fatal(err)
	}
	parentPath := "/system/democase/hangla_rank"
	formPath := parentPath + "/rate.form"
	if err := db.Create(&model.ServiceTree{
		Name:         "排行榜",
		Code:         "hangla_rank",
		Type:         model.ServiceTreeTypePackage,
		AppID:        appModel.ID,
		RefID:        2,
		FullCodePath: parentPath,
	}).Error; err != nil {
		t.Fatal(err)
	}

	created, err := requestService.CreateRequest(
		actorContext("carol"),
		"system",
		"democase",
		"carol",
		parentPath,
		access.RoleMember,
		"需要提交评分",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := requestService.Approve(actorContext("system"), created.ID, "system", "同意使用"); err != nil {
		t.Fatal(err)
	}

	if err := permission.RequirePermission(
		context.Background(),
		"system",
		"democase",
		"carol",
		formPath,
		access.ActionWrite,
	); err != nil {
		t.Fatalf("approved directory member should submit child form: %v", err)
	}
	if err := permission.RequirePermission(
		context.Background(),
		"system",
		"democase",
		"carol",
		parentPath,
		access.ActionAdmin,
	); err == nil {
		t.Fatal("approved member must not receive admin")
	}
}

func TestPermissionRequestAllowsInheritedMemberToUpgradeChildToAdmin(t *testing.T) {
	requestService, permission, _ := newPermissionRequestTestService(t)
	parentPath := "/alice/ops/tickets"
	resourcePath := "/alice/ops/tickets/submit.form"

	if err := permission.GrantRole(actorContext("bob"), access.GrantRoleRequest{
		TenantUser:   "alice",
		App:          "ops",
		Principal:    access.Principal{Type: access.PrincipalUser, Key: "carol"},
		ResourcePath: parentPath,
		RoleCode:     access.RoleMember,
		CreatedBy:    "bob",
	}); err != nil {
		t.Fatal(err)
	}

	resolved, err := permission.ResolvePermissions(context.Background(), "alice", "ops", "carol", resourcePath)
	if err != nil {
		t.Fatal(err)
	}
	if !permissionSetCoversRole(resolved.Permissions, access.RoleMember) || resolved.InheritedFrom != parentPath {
		t.Fatalf("resolved = %#v, want inherited member from %s", resolved, parentPath)
	}

	for _, roleCode := range []access.RoleCode{access.RoleViewer, access.RoleMember} {
		if _, err := requestService.CreateRequest(
			actorContext("carol"), "alice", "ops", "carol", resourcePath,
			roleCode, "申请已有或更低权限", nil,
		); err == nil || !strings.Contains(err.Error(), "无需重复申请") {
			t.Fatalf("request %s error = %v, want covered-role rejection", roleCode, err)
		}
	}
	if _, err := requestService.CreateRequest(
		actorContext("carol"), "alice", "ops", "carol", resourcePath,
		access.RoleOwner, "申请所有权", nil,
	); err == nil || !strings.Contains(err.Error(), "Viewer、Member 或 Admin") {
		t.Fatalf("owner request error = %v", err)
	}

	created, err := requestService.CreateRequest(
		actorContext("carol"), "alice", "ops", "carol", resourcePath,
		access.RoleAdmin, "需要管理当前函数", nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if created.RequestedRole != access.RoleAdmin {
		t.Fatalf("requested role = %q", created.RequestedRole)
	}
	if _, err := requestService.Approve(actorContext("bob"), created.ID, "bob", "同意升级"); err != nil {
		t.Fatal(err)
	}

	canAdmin, err := permission.HasPermission(context.Background(), "alice", "ops", "carol", resourcePath, access.ActionAdmin)
	if err != nil {
		t.Fatal(err)
	}
	if !canAdmin {
		t.Fatal("approved admin upgrade should grant admin permission on the child resource")
	}
}

func TestPermissionRequestRejectAndCancelDoNotGrantRole(t *testing.T) {
	requestService, permission, _ := newPermissionRequestTestService(t)
	notifier := &recordingPermissionNotifier{}
	permission.permissionNotifier = notifier
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
	if len(notifier.notifications) != 1 || notifier.notifications[0].ToUser != "carol" {
		t.Fatalf("rejection notifications = %#v", notifier.notifications)
	}
	requireNotificationContains(t, notifier.notifications[0], "权限申请已驳回", resourcePath, "用途不明确")

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
	if len(notifier.notifications) != 1 {
		t.Fatalf("cancel should not send another notification: %#v", notifier.notifications)
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
