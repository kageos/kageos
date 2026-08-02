package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	appmodel "github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/core/app-server/repository"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/access"
	"github.com/kageos/kageos/pkg/contextx"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newPermissionTestService(t *testing.T) (*PermissionService, *repository.AppRepository, *gorm.DB) {
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
	if err := db.AutoMigrate(&appmodel.App{}, &appmodel.WorkspaceRoleAssignment{}, &appmodel.OperateLog{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	appRepo := repository.NewAppRepository(db)
	if err := appRepo.CreateApp(&appmodel.App{User: "alice", Code: "ops", Name: "Ops", Version: "v1"}); err != nil {
		t.Fatalf("create app: %v", err)
	}
	service := NewPermissionService(
		repository.NewRoleAssignmentRepository(db),
		repository.NewOperateLogRepository(db),
		appRepo,
	)
	service.userLookup = func(ctx context.Context, username string) (*dto.UserInfo, error) {
		return &dto.UserInfo{Username: username}, nil
	}
	return service, appRepo, db
}

func actorContext(username string) context.Context {
	return context.WithValue(context.Background(), contextx.RequestUserHeader, username)
}

func userPrincipal(username string) access.Principal {
	return access.Principal{Type: access.PrincipalUser, Key: username}
}

func userPrincipals(usernames ...string) []access.Principal {
	principals := make([]access.Principal, 0, len(usernames))
	for _, username := range usernames {
		principals = append(principals, userPrincipal(username))
	}
	return principals
}

func departmentPrincipal(path string) access.Principal {
	return access.Principal{Type: access.PrincipalDepartment, Key: path}
}

func TestPermissionGrantAndResolveInheritedPermissions(t *testing.T) {
	service, _, _ := newPermissionTestService(t)
	ctx := actorContext("alice")

	if err := service.GrantRole(ctx, access.GrantRoleRequest{
		TenantUser:   "alice",
		App:          "ops",
		Principal:    userPrincipal("bob"),
		ResourcePath: "/alice/ops/ticket",
		RoleCode:     access.RoleMember,
		CreatedBy:    "alice",
	}); err != nil {
		t.Fatal(err)
	}

	result, err := service.ResolvePermissions(ctx, "alice", "ops", "bob", "/alice/ops/ticket/sub/items.table")
	if err != nil {
		t.Fatal(err)
	}
	if !access.HasPermission(result.Permissions, access.ActionUpdate) {
		t.Fatal("member should inherit update")
	}
	if access.HasPermission(result.Permissions, access.ActionDelete) {
		t.Fatal("member should not inherit delete")
	}
	if result.InheritedFrom != "/alice/ops/ticket" {
		t.Fatalf("inherited_from = %q", result.InheritedFrom)
	}
}

func TestPermissionExpiredAssignmentDoesNotGrantPermission(t *testing.T) {
	service, _, db := newPermissionTestService(t)
	expired := time.Now().Add(-time.Minute)
	assignment := &appmodel.WorkspaceRoleAssignment{
		TenantUser:    "alice",
		App:           "ops",
		PrincipalType: string(access.PrincipalUser),
		PrincipalKey:  "bob",
		ResourcePath:  "/alice/ops",
		RoleCode:      string(access.RoleAdmin),
		ExpiresAt:     &expired,
	}
	assignment.CreatedBy = "alice"
	if err := db.Create(assignment).Error; err != nil {
		t.Fatal(err)
	}

	ok, err := service.HasPermission(context.Background(), "alice", "ops", "bob", "/alice/ops/ticket", access.ActionAdmin)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expired admin assignment should not grant admin")
	}
}

func TestPermissionGrantRejectsExpiredAssignment(t *testing.T) {
	service, _, db := newPermissionTestService(t)
	expired := time.Now().Add(-time.Minute)
	err := service.GrantRole(actorContext("alice"), access.GrantRoleRequest{
		TenantUser:   "alice",
		App:          "ops",
		Principal:    userPrincipal("bob"),
		ResourcePath: "/alice/ops",
		RoleCode:     access.RoleMember,
		ExpiresAt:    &expired,
		CreatedBy:    "alice",
	})
	if err == nil || !strings.Contains(err.Error(), "权限到期时间必须晚于当前时间") {
		t.Fatalf("expected expired grant validation error, got %v", err)
	}

	var count int64
	if err := db.Model(&appmodel.WorkspaceRoleAssignment{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expired grant must not write an assignment, got %d", count)
	}
}

func TestPermissionOwnerFallback(t *testing.T) {
	service, _, _ := newPermissionTestService(t)
	ok, err := service.HasPermission(context.Background(), "alice", "ops", "alice", "/alice/ops/anything", access.ActionDelete)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("workspace owner should have delete")
	}
}

func TestPermissionSystemBuiltinRequiresExplicitRead(t *testing.T) {
	service, _, _ := newPermissionTestService(t)

	canRead, err := service.HasPermission(context.Background(), "system", "prompt", "alice", "/system/prompt/case_catalog/table/ticket", access.ActionRead)
	if err != nil {
		t.Fatal(err)
	}
	if canRead {
		t.Fatal("system workspace nodes must not synthesize read permission")
	}

	canWrite, err := service.HasPermission(context.Background(), "system", "prompt", "alice", "/system/prompt/case_catalog/table/ticket", access.ActionWrite)
	if err != nil {
		t.Fatal(err)
	}
	if canWrite {
		t.Fatal("system builtin resources should not grant write")
	}
}

func TestPermissionSystemBuiltinHonorsExplicitMemberForExecutionAndTree(t *testing.T) {
	service, _, db := newPermissionTestService(t)
	parentPath := "/system/democase/hangla_rank"
	formPath := parentPath + "/rate.form"
	assignment := &appmodel.WorkspaceRoleAssignment{
		TenantUser:    "system",
		App:           "democase",
		PrincipalType: string(access.PrincipalUser),
		PrincipalKey:  "bob",
		ResourcePath:  parentPath,
		RoleCode:      string(access.RoleMember),
	}
	assignment.CreatedBy = "system"
	if err := db.Create(assignment).Error; err != nil {
		t.Fatal(err)
	}

	resolved, err := service.ResolvePermissions(context.Background(), "system", "democase", "bob", formPath)
	if err != nil {
		t.Fatal(err)
	}
	if !access.HasPermission(resolved.Permissions, access.ActionWrite) {
		t.Fatalf("explicit member on system directory should grant inherited form write: %#v", resolved)
	}
	if access.HasPermission(resolved.Permissions, access.ActionAdmin) {
		t.Fatalf("member should not grant admin: %#v", resolved)
	}
	if resolved.InheritedFrom != parentPath {
		t.Fatalf("inherited_from = %q, want %q", resolved.InheritedFrom, parentPath)
	}
	if len(resolved.RoleCodes) != 1 || resolved.RoleCodes[0] != access.RoleMember {
		t.Fatalf("role_codes = %#v, want member without synthetic viewer", resolved.RoleCodes)
	}
	if err := service.RequirePermission(
		context.Background(),
		"system",
		"democase",
		"bob",
		formPath,
		access.ActionWrite,
	); err != nil {
		t.Fatalf("form execution should accept inherited system member: %v", err)
	}

	tree, err := service.PermissionsForTree(
		context.Background(),
		"system",
		"democase",
		"bob",
		[]string{parentPath, formPath, "/system/democase/other"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !access.HasPermission(tree[formPath].Permissions, access.ActionWrite) {
		t.Fatalf("tree resolver should agree with execution resolver: %#v", tree[formPath])
	}
	if access.HasPermission(tree["/system/democase/other"].Permissions, access.ActionRead) ||
		access.HasPermission(tree["/system/democase/other"].Permissions, access.ActionWrite) {
		t.Fatalf("unassigned system resource should remain locked: %#v", tree["/system/democase/other"])
	}
	if len(tree["/system/democase/other"].RoleCodes) != 0 {
		t.Fatalf("unassigned system resource should not expose a synthetic role: %#v", tree["/system/democase/other"])
	}
}

func TestPermissionSystemUserHasOwnerPermissionOnSystemBuiltin(t *testing.T) {
	service, _, _ := newPermissionTestService(t)

	result, err := service.ResolvePermissions(context.Background(), "system", "tools", "system", "/system/tools")
	if err != nil {
		t.Fatal(err)
	}
	if !access.HasPermission(result.Permissions, access.ActionAdmin) {
		t.Fatal("system user should have admin permission on /system/tools")
	}
	if !access.HasPermission(result.Permissions, access.ActionOwner) {
		t.Fatal("system user should have owner permission on /system/tools")
	}
	if err := service.RequirePermission(context.Background(), "system", "tools", "system", "/system/tools", access.ActionAdmin); err != nil {
		t.Fatalf("system user should pass admin check: %v", err)
	}
}

func TestPermissionAdminCanGrantMember(t *testing.T) {
	service, _, _ := newPermissionTestService(t)
	ctx := actorContext("alice")
	if err := service.GrantRole(ctx, access.GrantRoleRequest{
		TenantUser:   "alice",
		App:          "ops",
		Principal:    userPrincipal("admin-user"),
		ResourcePath: "/alice/ops",
		RoleCode:     access.RoleAdmin,
		CreatedBy:    "alice",
	}); err != nil {
		t.Fatal(err)
	}

	if err := service.GrantRole(actorContext("admin-user"), access.GrantRoleRequest{
		TenantUser:   "alice",
		App:          "ops",
		Principal:    userPrincipal("member-user"),
		ResourcePath: "/alice/ops/ticket",
		RoleCode:     access.RoleMember,
		CreatedBy:    "admin-user",
	}); err != nil {
		t.Fatal(err)
	}
}

func TestPermissionBatchGrantRolesGrantsEveryCombination(t *testing.T) {
	service, _, db := newPermissionTestService(t)
	notifier := &recordingPermissionNotifier{}
	service.permissionNotifier = notifier

	if err := service.BatchGrantRoles(actorContext("alice"), access.BatchGrantRoleRequest{
		TenantUser:    "alice",
		App:           "ops",
		Principals:    userPrincipals("bob", "cora"),
		ResourcePaths: []string{"/alice/ops/ticket", "/alice/ops/report"},
		RoleCodes:     []access.RoleCode{access.RoleViewer, access.RoleMember},
		CreatedBy:     "alice",
	}); err != nil {
		t.Fatal(err)
	}

	var count int64
	if err := db.Model(&appmodel.WorkspaceRoleAssignment{}).Count(&count).Error; err != nil {
		t.Fatalf("count assignments: %v", err)
	}
	if count != 8 {
		t.Fatalf("assignment count = %d, want 8", count)
	}

	for _, username := range []string{"bob", "cora"} {
		result, err := service.ResolvePermissions(context.Background(), "alice", "ops", username, "/alice/ops/ticket/sub/items.table")
		if err != nil {
			t.Fatal(err)
		}
		if !access.HasPermission(result.Permissions, access.ActionUpdate) {
			t.Fatalf("%s should have inherited member update permission: %#v", username, result)
		}
		if access.HasPermission(result.Permissions, access.ActionDelete) {
			t.Fatalf("%s should not receive delete permission: %#v", username, result)
		}
	}

	ok, err := service.HasPermission(context.Background(), "alice", "ops", "bob", "/alice/ops/other", access.ActionRead)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("batch assignment should not leak read permission to unrelated paths")
	}
	if len(notifier.notifications) != 2 {
		t.Fatalf("batch notifications = %#v, want one per user", notifier.notifications)
	}
	for _, notification := range notifier.notifications {
		if notification.ToUser != "bob" && notification.ToUser != "cora" {
			t.Fatalf("unexpected notification recipient: %#v", notification)
		}
		requireNotificationContains(t, notification, "你已获得新的权限", "/alice/ops/ticket", "/alice/ops/report", "查看者", "成员")
	}
}

func TestPermissionGrantNotifiesDirectUser(t *testing.T) {
	service, _, _ := newPermissionTestService(t)
	notifier := &recordingPermissionNotifier{}
	service.permissionNotifier = notifier

	if err := service.GrantRole(actorContext("alice"), accessGrantForNotificationTest("bob")); err != nil {
		t.Fatal(err)
	}
	if len(notifier.notifications) != 1 || notifier.notifications[0].ToUser != "bob" {
		t.Fatalf("notifications = %#v", notifier.notifications)
	}
	requireNotificationContains(t, notifier.notifications[0], "你已获得新的权限", "/alice/ops/ticket", "成员", "alice")
}

func TestPermissionBatchGrantRolesValidatesAllTargetsBeforeWriting(t *testing.T) {
	service, _, db := newPermissionTestService(t)
	service.userLookup = func(ctx context.Context, username string) (*dto.UserInfo, error) {
		if username == "missing-user" {
			return nil, errors.New("user not found")
		}
		return &dto.UserInfo{Username: username}, nil
	}

	err := service.BatchGrantRoles(actorContext("alice"), access.BatchGrantRoleRequest{
		TenantUser:    "alice",
		App:           "ops",
		Principals:    userPrincipals("bob", "missing-user"),
		ResourcePaths: []string{"/alice/ops/ticket"},
		RoleCodes:     []access.RoleCode{access.RoleMember},
		CreatedBy:     "alice",
	})
	if err == nil || !strings.Contains(err.Error(), "被授权用户不存在") {
		t.Fatalf("expected target validation error, got %v", err)
	}

	var count int64
	if err := db.Model(&appmodel.WorkspaceRoleAssignment{}).Count(&count).Error; err != nil {
		t.Fatalf("count assignments: %v", err)
	}
	if count != 0 {
		t.Fatalf("batch validation failure must not leave partial assignments, got %d", count)
	}
}

func TestPermissionBatchGrantRolesRejectsExpiredAssignmentBeforeWriting(t *testing.T) {
	service, _, db := newPermissionTestService(t)
	expired := time.Now().Add(-time.Minute)
	err := service.BatchGrantRoles(actorContext("alice"), access.BatchGrantRoleRequest{
		TenantUser:    "alice",
		App:           "ops",
		Principals:    userPrincipals("bob", "cora"),
		ResourcePaths: []string{"/alice/ops/ticket"},
		RoleCodes:     []access.RoleCode{access.RoleMember},
		ExpiresAt:     &expired,
		CreatedBy:     "alice",
	})
	if err == nil || !strings.Contains(err.Error(), "权限到期时间必须晚于当前时间") {
		t.Fatalf("expected expired batch validation error, got %v", err)
	}

	var count int64
	if err := db.Model(&appmodel.WorkspaceRoleAssignment{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expired batch must not write assignments, got %d", count)
	}
}

func TestPermissionBatchGrantRolesRollsBackWhenAWriteFails(t *testing.T) {
	service, _, db := newPermissionTestService(t)
	callbackName := "test:fail_batch_role_assignment"
	if err := db.Callback().Create().Before("gorm:create").Register(callbackName, func(tx *gorm.DB) {
		assignment, ok := tx.Statement.Dest.(*appmodel.WorkspaceRoleAssignment)
		if ok && assignment.PrincipalKey == "cora" {
			tx.AddError(errors.New("forced assignment write failure"))
		}
	}); err != nil {
		t.Fatalf("register create callback: %v", err)
	}
	defer db.Callback().Create().Remove(callbackName)

	err := service.BatchGrantRoles(actorContext("alice"), access.BatchGrantRoleRequest{
		TenantUser:    "alice",
		App:           "ops",
		Principals:    userPrincipals("bob", "cora"),
		ResourcePaths: []string{"/alice/ops/ticket"},
		RoleCodes:     []access.RoleCode{access.RoleMember},
		CreatedBy:     "alice",
	})
	if err == nil || !strings.Contains(err.Error(), "forced assignment write failure") {
		t.Fatalf("expected forced write failure, got %v", err)
	}

	var count int64
	if err := db.Model(&appmodel.WorkspaceRoleAssignment{}).Count(&count).Error; err != nil {
		t.Fatalf("count assignments: %v", err)
	}
	if count != 0 {
		t.Fatalf("batch write failure must roll back every assignment, got %d", count)
	}
}

func TestPermissionGrantValidatesTargetUser(t *testing.T) {
	service, _, _ := newPermissionTestService(t)
	ctx := contextx.WithRequestInfo(actorContext("alice"), contextx.RequestInfo{
		RequestUser: "alice",
	})
	lookups := 0
	service.userLookup = func(ctx context.Context, username string) (*dto.UserInfo, error) {
		lookups++
		if username != "member-user" {
			t.Fatalf("unexpected lookup username: %s", username)
		}
		return &dto.UserInfo{Username: username}, nil
	}

	if err := service.GrantRole(ctx, access.GrantRoleRequest{
		TenantUser:   "alice",
		App:          "ops",
		Principal:    userPrincipal("member-user"),
		ResourcePath: "/alice/ops/ticket",
		RoleCode:     access.RoleMember,
		CreatedBy:    "alice",
	}); err != nil {
		t.Fatal(err)
	}
	if lookups != 1 {
		t.Fatalf("expected one target user lookup, got %d", lookups)
	}
}

func TestPermissionGrantRejectsUnknownTargetUser(t *testing.T) {
	service, _, _ := newPermissionTestService(t)
	ctx := contextx.WithRequestInfo(actorContext("alice"), contextx.RequestInfo{
		RequestUser: "alice",
	})
	service.userLookup = func(ctx context.Context, username string) (*dto.UserInfo, error) {
		return nil, errors.New("user not found")
	}

	err := service.GrantRole(ctx, access.GrantRoleRequest{
		TenantUser:   "alice",
		App:          "ops",
		Principal:    userPrincipal("external-user"),
		ResourcePath: "/alice/ops/ticket",
		RoleCode:     access.RoleMember,
		CreatedBy:    "alice",
	})
	if err == nil || !strings.Contains(err.Error(), "被授权用户不存在") {
		t.Fatalf("expected target validation error, got %v", err)
	}
}

func TestPermissionMemberCannotGrant(t *testing.T) {
	service, _, _ := newPermissionTestService(t)
	ctx := actorContext("alice")
	if err := service.GrantRole(ctx, access.GrantRoleRequest{
		TenantUser:   "alice",
		App:          "ops",
		Principal:    userPrincipal("member-user"),
		ResourcePath: "/alice/ops",
		RoleCode:     access.RoleMember,
		CreatedBy:    "alice",
	}); err != nil {
		t.Fatal(err)
	}

	err := service.GrantRole(actorContext("member-user"), access.GrantRoleRequest{
		TenantUser:   "alice",
		App:          "ops",
		Principal:    userPrincipal("other-user"),
		ResourcePath: "/alice/ops/ticket",
		RoleCode:     access.RoleViewer,
		CreatedBy:    "member-user",
	})
	if err == nil {
		t.Fatal("member should not grant roles")
	}
}

func TestPermissionHasAnyWorkspacePermissionForChildGrant(t *testing.T) {
	service, _, _ := newPermissionTestService(t)
	if err := service.GrantRole(actorContext("alice"), access.GrantRoleRequest{
		TenantUser:   "alice",
		App:          "ops",
		Principal:    userPrincipal("child-user"),
		ResourcePath: "/alice/ops/ticket/sub",
		RoleCode:     access.RoleViewer,
		CreatedBy:    "alice",
	}); err != nil {
		t.Fatal(err)
	}

	ok, err := service.HasAnyWorkspacePermission(context.Background(), "alice", "ops", "child-user")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("child resource assignment should allow workspace tree entry")
	}
}

func TestPermissionListAccessibleApps(t *testing.T) {
	service, _, _ := newPermissionTestService(t)
	if err := service.GrantRole(actorContext("alice"), access.GrantRoleRequest{
		TenantUser:   "alice",
		App:          "ops",
		Principal:    userPrincipal("bob"),
		ResourcePath: "/alice/ops/ticket",
		RoleCode:     access.RoleViewer,
		CreatedBy:    "alice",
	}); err != nil {
		t.Fatal(err)
	}

	apps, err := service.ListAccessibleApps(context.Background(), "bob")
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 1 || apps[0].User != "alice" || apps[0].Code != "ops" {
		t.Fatalf("unexpected accessible apps: %+v", apps)
	}
}

func TestPermissionOrganizationAndResourceInheritance(t *testing.T) {
	service, _, _ := newPermissionTestService(t)
	service.departmentLookup = func(ctx context.Context, departmentPath string) (bool, error) {
		return departmentPath == "/org" || departmentPath == "/org/sales", nil
	}
	service.userLookup = func(ctx context.Context, username string) (*dto.UserInfo, error) {
		departmentPath := "/org/finance"
		if username == "bob" {
			departmentPath = "/org/sales/east"
		}
		return &dto.UserInfo{Username: username, DepartmentFullPath: departmentPath}, nil
	}

	if err := service.GrantRole(actorContext("alice"), access.GrantRoleRequest{
		TenantUser:   "alice",
		App:          "ops",
		Principal:    departmentPrincipal("/org"),
		ResourcePath: "/alice/ops",
		RoleCode:     access.RoleViewer,
		CreatedBy:    "alice",
	}); err != nil {
		t.Fatal(err)
	}
	if err := service.GrantRole(actorContext("alice"), access.GrantRoleRequest{
		TenantUser:   "alice",
		App:          "ops",
		Principal:    departmentPrincipal("/org/sales"),
		ResourcePath: "/alice/ops/crm",
		RoleCode:     access.RoleMember,
		CreatedBy:    "alice",
	}); err != nil {
		t.Fatal(err)
	}

	bob, err := service.ResolvePermissions(context.Background(), "alice", "ops", "bob", "/alice/ops/crm/leads.table")
	if err != nil {
		t.Fatal(err)
	}
	if !access.HasPermission(bob.Permissions, access.ActionRead) || !access.HasPermission(bob.Permissions, access.ActionUpdate) {
		t.Fatalf("sales descendant should inherit org read and sales update: %#v", bob)
	}

	cora, err := service.ResolvePermissions(context.Background(), "alice", "ops", "cora", "/alice/ops/crm/leads.table")
	if err != nil {
		t.Fatal(err)
	}
	if !access.HasPermission(cora.Permissions, access.ActionRead) {
		t.Fatalf("all organization members should inherit /org read: %#v", cora)
	}
	if access.HasPermission(cora.Permissions, access.ActionUpdate) {
		t.Fatalf("non-sales member must not inherit sales update: %#v", cora)
	}
}

func TestPermissionOrganizationGrantMakesWorkspaceDiscoverable(t *testing.T) {
	service, _, _ := newPermissionTestService(t)
	service.departmentLookup = func(ctx context.Context, departmentPath string) (bool, error) {
		return departmentPath == "/org", nil
	}
	service.userLookup = func(ctx context.Context, username string) (*dto.UserInfo, error) {
		return &dto.UserInfo{Username: username, DepartmentFullPath: "/org/unassigned"}, nil
	}
	if err := service.GrantRole(actorContext("alice"), access.GrantRoleRequest{
		TenantUser:   "alice",
		App:          "ops",
		Principal:    departmentPrincipal("/org"),
		ResourcePath: "/alice/ops",
		RoleCode:     access.RoleViewer,
		CreatedBy:    "alice",
	}); err != nil {
		t.Fatal(err)
	}

	apps, err := service.ListAccessibleApps(context.Background(), "new-user")
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 1 || apps[0].User != "alice" || apps[0].Code != "ops" {
		t.Fatalf("organization grant should expose workspace to new members: %+v", apps)
	}
}

func TestPermissionGrantRejectsUnknownOrganization(t *testing.T) {
	service, _, _ := newPermissionTestService(t)
	service.departmentLookup = func(ctx context.Context, departmentPath string) (bool, error) {
		return false, nil
	}

	err := service.GrantRole(actorContext("alice"), access.GrantRoleRequest{
		TenantUser:   "alice",
		App:          "ops",
		Principal:    departmentPrincipal("/org/missing"),
		ResourcePath: "/alice/ops",
		RoleCode:     access.RoleViewer,
		CreatedBy:    "alice",
	})
	if err == nil || !strings.Contains(err.Error(), "被授权组织不存在") {
		t.Fatalf("expected organization validation error, got %v", err)
	}
}

func TestPermissionListAssignmentsSeparatesCurrentAndInherited(t *testing.T) {
	service, _, _ := newPermissionTestService(t)
	ctx := actorContext("alice")
	if err := service.GrantRole(ctx, access.GrantRoleRequest{
		TenantUser:   "alice",
		App:          "ops",
		Principal:    userPrincipal("parent-user"),
		ResourcePath: "/alice/ops/ticket",
		RoleCode:     access.RoleViewer,
		CreatedBy:    "alice",
	}); err != nil {
		t.Fatal(err)
	}
	if err := service.GrantRole(ctx, access.GrantRoleRequest{
		TenantUser:   "alice",
		App:          "ops",
		Principal:    userPrincipal("current-user"),
		ResourcePath: "/alice/ops/ticket/sub",
		RoleCode:     access.RoleMember,
		CreatedBy:    "alice",
	}); err != nil {
		t.Fatal(err)
	}
	if err := service.GrantRole(ctx, access.GrantRoleRequest{
		TenantUser:   "alice",
		App:          "ops",
		Principal:    userPrincipal("sibling-user"),
		ResourcePath: "/alice/ops/other",
		RoleCode:     access.RoleAdmin,
		CreatedBy:    "alice",
	}); err != nil {
		t.Fatal(err)
	}

	members, err := service.ListAssignments(ctx, "alice", "ops", "/alice/ops/ticket/sub")
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 2 {
		t.Fatalf("expected 2 effective members, got %+v", members)
	}

	var inherited, current *access.RoleAssignmentView
	for i := range members {
		member := &members[i]
		switch member.PrincipalKey {
		case "parent-user":
			inherited = member
		case "current-user":
			current = member
		}
	}
	if inherited == nil || inherited.Source != "inherited" || inherited.Direct || inherited.InheritedFrom != "/alice/ops/ticket" {
		t.Fatalf("unexpected inherited member: %+v", inherited)
	}
	if current == nil || current.Source != "current" || !current.Direct || current.InheritedFrom != "" {
		t.Fatalf("unexpected current member: %+v", current)
	}
}

func TestPermissionGrantWritesOperateLog(t *testing.T) {
	service, _, db := newPermissionTestService(t)
	ctx := contextx.WithRequestInfo(actorContext("alice"), contextx.RequestInfo{
		SourceType: contextx.SourceTypeOpenAPIToken,
		SourceRef:  "alice",
	})
	service.userLookup = func(ctx context.Context, username string) (*dto.UserInfo, error) {
		return &dto.UserInfo{Username: username}, nil
	}

	if err := service.GrantRole(ctx, access.GrantRoleRequest{
		TenantUser:   "alice",
		App:          "ops",
		Principal:    userPrincipal("bob"),
		ResourcePath: "/alice/ops/ticket",
		RoleCode:     access.RoleMember,
		CreatedBy:    "alice",
	}); err != nil {
		t.Fatal(err)
	}

	log := waitOperateLog(t, db, "permission.role.granted")
	if log.ActorUser != "alice" || log.TargetUser != "bob" || log.ResourcePath != "/alice/ops/ticket" {
		t.Fatalf("unexpected assign log: %+v", log)
	}
	if log.ResourceType != "permission" || log.Status != "success" {
		t.Fatalf("unexpected assign log type/status: %+v", log)
	}
	if log.Source != contextx.ClientSourceOpenAPI {
		t.Fatalf("source = %q, want openapi", log.Source)
	}

	var newValues dto.PermissionRoleGrantedValues
	if err := json.Unmarshal(log.NewValuesJSON, &newValues); err != nil {
		t.Fatalf("unmarshal assign new values: %v", err)
	}
	if newValues.RoleCode != string(access.RoleMember) {
		t.Fatalf("role_code = %q", newValues.RoleCode)
	}
}

func TestPermissionRevokeWritesOperateLog(t *testing.T) {
	service, _, db := newPermissionTestService(t)
	ctx := actorContext("alice")

	if err := service.GrantRole(ctx, access.GrantRoleRequest{
		TenantUser:   "alice",
		App:          "ops",
		Principal:    userPrincipal("bob"),
		ResourcePath: "/alice/ops/ticket",
		RoleCode:     access.RoleViewer,
		CreatedBy:    "alice",
	}); err != nil {
		t.Fatal(err)
	}
	waitOperateLog(t, db, "permission.role.granted")

	if err := service.RevokeRole(ctx, access.RevokeRoleRequest{
		TenantUser:   "alice",
		App:          "ops",
		Principal:    userPrincipal("bob"),
		ResourcePath: "/alice/ops/ticket",
		RoleCode:     access.RoleViewer,
		Actor:        "alice",
	}); err != nil {
		t.Fatal(err)
	}

	log := waitOperateLog(t, db, "permission.role.revoked")
	if log.ActorUser != "alice" || log.TargetUser != "bob" || log.ResourcePath != "/alice/ops/ticket" {
		t.Fatalf("unexpected remove log: %+v", log)
	}

	var details dto.PermissionRoleRevokedDetails
	if err := json.Unmarshal(log.DetailsJSON, &details); err != nil {
		t.Fatalf("unmarshal remove details: %v", err)
	}
	if details.RoleCode != string(access.RoleViewer) || details.RowsAffected != 1 {
		t.Fatalf("unexpected remove details: %+v", details)
	}
}

func waitOperateLog(t *testing.T, db *gorm.DB, action string) appmodel.OperateLog {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	var log appmodel.OperateLog
	for {
		err := db.Where("action = ?", action).Order("id DESC").First(&log).Error
		if err == nil {
			return log
		}
		if time.Now().After(deadline) {
			t.Fatalf("operate log %s not found: %v", action, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
