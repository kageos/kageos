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
	ctx := contextx.WithRequestInfo(actorContext("alice"), contextx.RequestInfo{CompanyCode: "acme"})
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
