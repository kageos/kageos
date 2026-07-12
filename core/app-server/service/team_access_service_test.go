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

func newTeamAccessTestService(t *testing.T) (*TeamAccessService, *repository.AppRepository, *gorm.DB) {
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
	return NewTeamAccessService(
		repository.NewTeamAccessRepository(db),
		repository.NewOperateLogRepository(db),
		appRepo,
	), appRepo, db
}

func actorContext(username string) context.Context {
	return context.WithValue(context.Background(), contextx.RequestUserHeader, username)
}

func TestTeamAccessAssignAndResolveInheritedPermissions(t *testing.T) {
	service, _, _ := newTeamAccessTestService(t)
	ctx := actorContext("alice")

	if err := service.Assign(ctx, access.AssignRoleRequest{
		TenantUser:   "alice",
		App:          "ops",
		Username:     "bob",
		ResourcePath: "/alice/ops/ticket",
		RoleCode:     access.RoleMember,
		CreatedBy:    "alice",
	}); err != nil {
		t.Fatal(err)
	}

	result, err := service.Resolve(ctx, "alice", "ops", "bob", "/alice/ops/ticket/sub/items.table")
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

func TestTeamAccessExpiredAssignmentDoesNotGrantPermission(t *testing.T) {
	service, _, _ := newTeamAccessTestService(t)
	expired := time.Now().Add(-time.Minute)
	if err := service.Assign(actorContext("alice"), access.AssignRoleRequest{
		TenantUser:   "alice",
		App:          "ops",
		Username:     "bob",
		ResourcePath: "/alice/ops",
		RoleCode:     access.RoleAdmin,
		ExpiresAt:    &expired,
		CreatedBy:    "alice",
	}); err != nil {
		t.Fatal(err)
	}

	ok, err := service.Can(context.Background(), "alice", "ops", "bob", "/alice/ops/ticket", access.ActionAdmin)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expired admin assignment should not grant admin")
	}
}

func TestTeamAccessOwnerFallback(t *testing.T) {
	service, _, _ := newTeamAccessTestService(t)
	ok, err := service.Can(context.Background(), "alice", "ops", "alice", "/alice/ops/anything", access.ActionDelete)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("workspace owner should have delete")
	}
}

func TestTeamAccessSystemBuiltinAllowsReadOnly(t *testing.T) {
	service, _, _ := newTeamAccessTestService(t)

	canRead, err := service.Can(context.Background(), "system", "prompt", "alice", "/system/prompt/case_catalog/table/ticket", access.ActionRead)
	if err != nil {
		t.Fatal(err)
	}
	if !canRead {
		t.Fatal("system builtin resources should be readable")
	}

	canWrite, err := service.Can(context.Background(), "system", "prompt", "alice", "/system/prompt/case_catalog/table/ticket", access.ActionWrite)
	if err != nil {
		t.Fatal(err)
	}
	if canWrite {
		t.Fatal("system builtin resources should not grant write")
	}
}

func TestTeamAccessSystemUserHasOwnerPermissionOnSystemBuiltin(t *testing.T) {
	service, _, _ := newTeamAccessTestService(t)

	result, err := service.Resolve(context.Background(), "system", "tools", "system", "/system/tools")
	if err != nil {
		t.Fatal(err)
	}
	if !access.HasPermission(result.Permissions, access.ActionAdmin) {
		t.Fatal("system user should have admin permission on /system/tools")
	}
	if !access.HasPermission(result.Permissions, access.ActionOwner) {
		t.Fatal("system user should have owner permission on /system/tools")
	}
	if err := service.Check(context.Background(), "system", "tools", "system", "/system/tools", access.ActionAdmin); err != nil {
		t.Fatalf("system user should pass admin check: %v", err)
	}
}

func TestTeamAccessAdminCanGrantMember(t *testing.T) {
	service, _, _ := newTeamAccessTestService(t)
	ctx := actorContext("alice")
	if err := service.Assign(ctx, access.AssignRoleRequest{
		TenantUser:   "alice",
		App:          "ops",
		Username:     "admin-user",
		ResourcePath: "/alice/ops",
		RoleCode:     access.RoleAdmin,
		CreatedBy:    "alice",
	}); err != nil {
		t.Fatal(err)
	}

	if err := service.Assign(actorContext("admin-user"), access.AssignRoleRequest{
		TenantUser:   "alice",
		App:          "ops",
		Username:     "member-user",
		ResourcePath: "/alice/ops/ticket",
		RoleCode:     access.RoleMember,
		CreatedBy:    "admin-user",
	}); err != nil {
		t.Fatal(err)
	}
}

func TestTeamAccessBatchAssignGrantsEveryCombination(t *testing.T) {
	service, _, db := newTeamAccessTestService(t)

	if err := service.BatchAssign(actorContext("alice"), access.BatchAssignRoleRequest{
		TenantUser:    "alice",
		App:           "ops",
		Usernames:     []string{"bob", "cora"},
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
		result, err := service.Resolve(context.Background(), "alice", "ops", username, "/alice/ops/ticket/sub/items.table")
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

	ok, err := service.Can(context.Background(), "alice", "ops", "bob", "/alice/ops/other", access.ActionRead)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("batch assignment should not leak read permission to unrelated paths")
	}
}

func TestTeamAccessAssignValidatesTargetUserWhenCompanyContextExists(t *testing.T) {
	service, _, _ := newTeamAccessTestService(t)
	ctx := contextx.WithRequestInfo(actorContext("alice"), contextx.RequestInfo{
		RequestUser: "alice",
		CompanyCode: "acme",
	})
	lookups := 0
	service.userLookup = func(ctx context.Context, username string) (*dto.UserInfo, error) {
		lookups++
		if username != "member-user" {
			t.Fatalf("unexpected lookup username: %s", username)
		}
		return &dto.UserInfo{Username: username, CompanyCode: "acme"}, nil
	}

	if err := service.Assign(ctx, access.AssignRoleRequest{
		TenantUser:   "alice",
		App:          "ops",
		Username:     "member-user",
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

func TestTeamAccessAssignRejectsUnknownTargetUserWhenCompanyContextExists(t *testing.T) {
	service, _, _ := newTeamAccessTestService(t)
	ctx := contextx.WithRequestInfo(actorContext("alice"), contextx.RequestInfo{
		RequestUser: "alice",
		CompanyCode: "acme",
	})
	service.userLookup = func(ctx context.Context, username string) (*dto.UserInfo, error) {
		return nil, errors.New("user not found")
	}

	err := service.Assign(ctx, access.AssignRoleRequest{
		TenantUser:   "alice",
		App:          "ops",
		Username:     "external-user",
		ResourcePath: "/alice/ops/ticket",
		RoleCode:     access.RoleMember,
		CreatedBy:    "alice",
	})
	if err == nil || !strings.Contains(err.Error(), "被授权用户不存在或不属于当前企业") {
		t.Fatalf("expected company target validation error, got %v", err)
	}
}

func TestTeamAccessMemberCannotGrant(t *testing.T) {
	service, _, _ := newTeamAccessTestService(t)
	ctx := actorContext("alice")
	if err := service.Assign(ctx, access.AssignRoleRequest{
		TenantUser:   "alice",
		App:          "ops",
		Username:     "member-user",
		ResourcePath: "/alice/ops",
		RoleCode:     access.RoleMember,
		CreatedBy:    "alice",
	}); err != nil {
		t.Fatal(err)
	}

	err := service.Assign(actorContext("member-user"), access.AssignRoleRequest{
		TenantUser:   "alice",
		App:          "ops",
		Username:     "other-user",
		ResourcePath: "/alice/ops/ticket",
		RoleCode:     access.RoleViewer,
		CreatedBy:    "member-user",
	})
	if err == nil {
		t.Fatal("member should not grant roles")
	}
}

func TestTeamAccessHasAnyWorkspaceAccessForChildGrant(t *testing.T) {
	service, _, _ := newTeamAccessTestService(t)
	if err := service.Assign(actorContext("alice"), access.AssignRoleRequest{
		TenantUser:   "alice",
		App:          "ops",
		Username:     "child-user",
		ResourcePath: "/alice/ops/ticket/sub",
		RoleCode:     access.RoleViewer,
		CreatedBy:    "alice",
	}); err != nil {
		t.Fatal(err)
	}

	ok, err := service.HasAnyWorkspaceAccess(context.Background(), "alice", "ops", "child-user")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("child resource assignment should allow workspace tree entry")
	}
}

func TestTeamAccessListAccessibleApps(t *testing.T) {
	service, _, _ := newTeamAccessTestService(t)
	if err := service.Assign(actorContext("alice"), access.AssignRoleRequest{
		TenantUser:   "alice",
		App:          "ops",
		Username:     "bob",
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

func TestOpenCollaborationWorkspaceGrantsAuthenticatedDataAccessOnly(t *testing.T) {
	service, appRepo, _ := newTeamAccessTestService(t)
	appModel, err := appRepo.GetAppByUserName("alice", "ops")
	if err != nil {
		t.Fatal(err)
	}
	appModel.AccessMode = appmodel.AppAccessModeOpenCollaboration
	if err := appRepo.UpdateApp(appModel); err != nil {
		t.Fatal(err)
	}

	for _, action := range []access.Action{access.ActionRead, access.ActionWrite, access.ActionUpdate} {
		ok, err := service.CanWorkspaceData(context.Background(), "alice", "ops", "bob", "/alice/ops/tickets.table", action)
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Fatalf("open collaboration should grant %s", action)
		}
	}

	for _, action := range []access.Action{access.ActionDelete, access.ActionAdmin, access.ActionOwner} {
		ok, err := service.CanWorkspaceData(context.Background(), "alice", "ops", "bob", "/alice/ops/tickets.table", action)
		if err != nil {
			t.Fatal(err)
		}
		if ok {
			t.Fatalf("open collaboration must not grant %s", action)
		}
	}

	// The original control-plane authorization remains unchanged.
	if ok, err := service.Can(context.Background(), "alice", "ops", "bob", "/alice/ops", access.ActionUpdate); err != nil {
		t.Fatal(err)
	} else if ok {
		t.Fatal("open collaboration must not alter explicit control-plane permissions")
	}

	if ok, err := service.HasAnyWorkspaceAccess(context.Background(), "alice", "ops", "bob"); err != nil {
		t.Fatal(err)
	} else if !ok {
		t.Fatal("open collaboration workspace should be discoverable by authenticated users")
	}

	permissions, err := service.PermissionsForTree(context.Background(), "alice", "ops", "bob", []string{"/alice/ops/tickets.table"})
	if err != nil {
		t.Fatal(err)
	}
	result := permissions["/alice/ops/tickets.table"]
	if result == nil || !access.HasPermission(result.Permissions, access.ActionUpdate) || access.HasPermission(result.Permissions, access.ActionDelete) {
		t.Fatalf("unexpected open collaboration tree permissions: %+v", result)
	}
}

func TestOpenCollaborationWorkspaceAppearsWithoutAssignment(t *testing.T) {
	service, appRepo, _ := newTeamAccessTestService(t)
	appModel, err := appRepo.GetAppByUserName("alice", "ops")
	if err != nil {
		t.Fatal(err)
	}
	appModel.AccessMode = appmodel.AppAccessModeOpenCollaboration
	if err := appRepo.UpdateApp(appModel); err != nil {
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

func TestTeamAccessListMembersSeparatesCurrentAndInherited(t *testing.T) {
	service, _, _ := newTeamAccessTestService(t)
	ctx := actorContext("alice")
	if err := service.Assign(ctx, access.AssignRoleRequest{
		TenantUser:   "alice",
		App:          "ops",
		Username:     "parent-user",
		ResourcePath: "/alice/ops/ticket",
		RoleCode:     access.RoleViewer,
		CreatedBy:    "alice",
	}); err != nil {
		t.Fatal(err)
	}
	if err := service.Assign(ctx, access.AssignRoleRequest{
		TenantUser:   "alice",
		App:          "ops",
		Username:     "current-user",
		ResourcePath: "/alice/ops/ticket/sub",
		RoleCode:     access.RoleMember,
		CreatedBy:    "alice",
	}); err != nil {
		t.Fatal(err)
	}
	if err := service.Assign(ctx, access.AssignRoleRequest{
		TenantUser:   "alice",
		App:          "ops",
		Username:     "sibling-user",
		ResourcePath: "/alice/ops/other",
		RoleCode:     access.RoleAdmin,
		CreatedBy:    "alice",
	}); err != nil {
		t.Fatal(err)
	}

	members, err := service.ListMembers(ctx, "alice", "ops", "/alice/ops/ticket/sub")
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 2 {
		t.Fatalf("expected 2 effective members, got %+v", members)
	}

	var inherited, current *access.MemberAccess
	for i := range members {
		member := &members[i]
		switch member.Username {
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

func TestTeamAccessAssignWritesOperateLog(t *testing.T) {
	service, _, db := newTeamAccessTestService(t)
	ctx := contextx.WithRequestInfo(actorContext("alice"), contextx.RequestInfo{
		CompanyCode: "acme",
		SourceType:  contextx.SourceTypeOpenAPIToken,
		SourceRef:   "alice",
	})
	service.userLookup = func(ctx context.Context, username string) (*dto.UserInfo, error) {
		return &dto.UserInfo{Username: username, CompanyCode: "acme"}, nil
	}

	if err := service.Assign(ctx, access.AssignRoleRequest{
		TenantUser:   "alice",
		App:          "ops",
		Username:     "bob",
		ResourcePath: "/alice/ops/ticket",
		RoleCode:     access.RoleMember,
		CreatedBy:    "alice",
	}); err != nil {
		t.Fatal(err)
	}

	log := waitOperateLog(t, db, "team.role.assigned")
	if log.ActorUser != "alice" || log.TargetUser != "bob" || log.ResourcePath != "/alice/ops/ticket" {
		t.Fatalf("unexpected assign log: %+v", log)
	}
	if log.ResourceType != "team_access" || log.Status != "success" {
		t.Fatalf("unexpected assign log type/status: %+v", log)
	}
	if log.CompanyCode != "acme" {
		t.Fatalf("company_code = %q", log.CompanyCode)
	}
	if log.Source != contextx.ClientSourceOpenAPI {
		t.Fatalf("source = %q, want openapi", log.Source)
	}

	var newValues dto.TeamRoleAssignedValues
	if err := json.Unmarshal(log.NewValuesJSON, &newValues); err != nil {
		t.Fatalf("unmarshal assign new values: %v", err)
	}
	if newValues.RoleCode != string(access.RoleMember) {
		t.Fatalf("role_code = %q", newValues.RoleCode)
	}
}

func TestTeamAccessRemoveWritesOperateLog(t *testing.T) {
	service, _, db := newTeamAccessTestService(t)
	ctx := actorContext("alice")

	if err := service.Assign(ctx, access.AssignRoleRequest{
		TenantUser:   "alice",
		App:          "ops",
		Username:     "bob",
		ResourcePath: "/alice/ops/ticket",
		RoleCode:     access.RoleViewer,
		CreatedBy:    "alice",
	}); err != nil {
		t.Fatal(err)
	}
	waitOperateLog(t, db, "team.role.assigned")

	if err := service.Remove(ctx, access.RemoveRoleRequest{
		TenantUser:   "alice",
		App:          "ops",
		Username:     "bob",
		ResourcePath: "/alice/ops/ticket",
		RoleCode:     access.RoleViewer,
		Actor:        "alice",
	}); err != nil {
		t.Fatal(err)
	}

	log := waitOperateLog(t, db, "team.role.removed")
	if log.ActorUser != "alice" || log.TargetUser != "bob" || log.ResourcePath != "/alice/ops/ticket" {
		t.Fatalf("unexpected remove log: %+v", log)
	}

	var details dto.TeamRoleRemovedDetails
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
