package service

import (
	"context"
	"strings"
	"testing"
	"time"

	appmodel "github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/pkg/access"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTeamAccessTestService(t *testing.T) (*TeamAccessService, *repository.AppRepository) {
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
	), appRepo
}

func actorContext(username string) context.Context {
	return context.WithValue(context.Background(), contextx.RequestUserHeader, username)
}

func TestTeamAccessAssignAndResolveInheritedPermissions(t *testing.T) {
	service, _ := newTeamAccessTestService(t)
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
	service, _ := newTeamAccessTestService(t)
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
	service, _ := newTeamAccessTestService(t)
	ok, err := service.Can(context.Background(), "alice", "ops", "alice", "/alice/ops/anything", access.ActionDelete)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("workspace owner should have delete")
	}
}

func TestTeamAccessAdminCanGrantMember(t *testing.T) {
	service, _ := newTeamAccessTestService(t)
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

func TestTeamAccessMemberCannotGrant(t *testing.T) {
	service, _ := newTeamAccessTestService(t)
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
	service, _ := newTeamAccessTestService(t)
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
	service, _ := newTeamAccessTestService(t)
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
	service, _ := newTeamAccessTestService(t)
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
