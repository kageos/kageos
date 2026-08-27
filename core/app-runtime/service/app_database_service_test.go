package service

import (
	"context"
	"strings"
	"testing"

	"github.com/kageos/kageos/core/app-runtime/model"
	"github.com/kageos/kageos/dto"
	appconfig "github.com/kageos/kageos/pkg/config"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestAppDatabaseService(t *testing.T) *AppDatabaseService {
	t.Helper()
	svc, err := NewAppDatabaseService(nil, appconfig.AppDatabaseConfig{
		Enabled:       true,
		Dialect:       "mysql",
		Host:          "127.0.0.1",
		Port:          3306,
		AdminUser:     "root",
		AdminPassword: "password",
		SecretKey:     "test-secret",
	})
	if err != nil {
		t.Fatalf("NewAppDatabaseService: %v", err)
	}
	return svc
}

func TestAppDatabaseRuntimePrivilegesAreNonDestructive(t *testing.T) {
	for _, forbidden := range []string{"DELETE", "CREATE", "ALTER", "DROP", "TRUNCATE", "INDEX", "REFERENCES"} {
		if strings.Contains(appDBRuntimePrivileges, forbidden) {
			t.Fatalf("runtime privileges must not include %s: %s", forbidden, appDBRuntimePrivileges)
		}
	}
	for _, required := range []string{"SELECT", "INSERT", "UPDATE"} {
		if !strings.Contains(appDBRuntimePrivileges, required) {
			t.Fatalf("runtime privileges missing %s: %s", required, appDBRuntimePrivileges)
		}
	}
}

func TestAppDatabaseMigrationPrivilegesExcludeDeleteAndDrop(t *testing.T) {
	for _, forbidden := range []string{"DELETE", "DROP", "TRUNCATE", "REFERENCES"} {
		if strings.Contains(appDBMigrationPrivileges, forbidden) {
			t.Fatalf("migration privileges must not include %s: %s", forbidden, appDBMigrationPrivileges)
		}
	}
	for _, required := range []string{"SELECT", "CREATE", "ALTER", "INDEX"} {
		if !strings.Contains(appDBMigrationPrivileges, required) {
			t.Fatalf("migration privileges missing %s: %s", required, appDBMigrationPrivileges)
		}
	}
}

func TestSoftDeleteCleanupDefaultsAreSafe(t *testing.T) {
	cfg := (appconfig.AppDatabaseConfig{}).WithDefaults().SoftDeleteCleanup
	if cfg.Enabled {
		t.Fatal("soft-delete cleanup must be disabled by default")
	}
	if cfg.Mode != "dry_run" {
		t.Fatalf("cleanup mode = %q, want dry_run", cfg.Mode)
	}
	if cfg.RetentionDays != 30 || cfg.IntervalMinutes != 1440 || cfg.BatchSize != 500 {
		t.Fatalf("unexpected cleanup defaults: %+v", cfg)
	}
}

func TestSoftDeleteCleanupRejectsUnsafeModeAndCapsBatch(t *testing.T) {
	cfg := (appconfig.AppDatabaseConfig{SoftDeleteCleanup: appconfig.SoftDeleteCleanupConfig{
		Enabled:   true,
		Mode:      "delete_everything",
		BatchSize: 50000,
	}}).WithDefaults().SoftDeleteCleanup
	if cfg.Mode != "dry_run" {
		t.Fatalf("invalid cleanup mode = %q, want dry_run", cfg.Mode)
	}
	if cfg.BatchSize != 10000 {
		t.Fatalf("cleanup batch = %d, want capped at 10000", cfg.BatchSize)
	}
}

func newCleanupPolicyTestService(t *testing.T) *AppDatabaseService {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open cleanup policy db: %v", err)
	}
	if err := model.InitTables(db); err != nil {
		t.Fatalf("migrate cleanup policy db: %v", err)
	}
	if err := db.Create(&model.AppDatabase{
		User: "alice", App: "crm", PackagePath: "sales", FullCodePath: "/alice/crm/sales",
		ClusterKey: "test", DatabaseName: "app_test", DatabaseUser: "app_user",
		PasswordCiphertext: "cipher", PasswordNonce: "nonce", Dialect: "mysql", Status: appDBStatusActive,
	}).Error; err != nil {
		t.Fatalf("create app database: %v", err)
	}
	svc, err := NewAppDatabaseService(db, appconfig.AppDatabaseConfig{
		Enabled: true, Dialect: "mysql", AdminUser: "root", AdminPassword: "password", SecretKey: "test-secret",
		SoftDeleteCleanup: appconfig.SoftDeleteCleanupConfig{
			Enabled: false, Mode: "dry_run", RetentionDays: 30, IntervalMinutes: 1440, BatchSize: 500,
		},
	})
	if err != nil {
		t.Fatalf("NewAppDatabaseService: %v", err)
	}
	return svc
}

func TestSoftDeleteCleanupPolicyUsesDeploymentDefaultAndPersistsTableOverride(t *testing.T) {
	svc := newCleanupPolicyTestService(t)
	scope := dto.AppDBCleanupPolicyReq{User: "alice", App: "crm", PackagePath: "sales", Table: "leads"}
	initial, err := svc.GetSoftDeleteCleanupPolicy(context.Background(), &scope)
	if err != nil {
		t.Fatalf("GetSoftDeleteCleanupPolicy default: %v", err)
	}
	if initial.Source != "deployment" || initial.Enabled || initial.RetentionDays != 30 {
		t.Fatalf("unexpected deployment policy: %+v", initial)
	}

	updated, err := svc.UpdateSoftDeleteCleanupPolicy(context.Background(), &dto.AppDBCleanupPolicyUpdateReq{
		AppDBCleanupPolicyReq: scope, Enabled: true, Mode: "purge", RetentionDays: 45,
	})
	if err != nil {
		t.Fatalf("UpdateSoftDeleteCleanupPolicy: %v", err)
	}
	if updated.Source != "table" || !updated.Enabled || updated.Mode != "purge" || updated.RetentionDays != 45 {
		t.Fatalf("unexpected table policy: %+v", updated)
	}
}

func TestSoftDeleteCleanupPolicyRejectsUnsafeValues(t *testing.T) {
	svc := newCleanupPolicyTestService(t)
	base := dto.AppDBCleanupPolicyReq{User: "alice", App: "crm", PackagePath: "sales", Table: "leads"}
	for _, request := range []*dto.AppDBCleanupPolicyUpdateReq{
		{AppDBCleanupPolicyReq: base, Enabled: true, Mode: "drop", RetentionDays: 30},
		{AppDBCleanupPolicyReq: base, Enabled: true, Mode: "purge", RetentionDays: 0},
	} {
		if _, err := svc.UpdateSoftDeleteCleanupPolicy(context.Background(), request); err == nil {
			t.Fatalf("unsafe cleanup policy should fail: %+v", request)
		}
	}
}

func TestAppDatabaseCapabilityIsScopedToRouterPackage(t *testing.T) {
	svc := newTestAppDatabaseService(t)
	capability, err := svc.IssueCapability("alice", "crm", "v1", "/sales/leads/list")
	if err != nil {
		t.Fatalf("IssueCapability: %v", err)
	}

	if err := svc.validateCapability(&dto.AppDBResolveReq{
		Capability:  capability,
		User:        "alice",
		App:         "crm",
		Version:     "v1",
		PackagePath: "sales/leads",
		Access:      dto.AppDBAccessRuntime,
	}); err != nil {
		t.Fatalf("same package should validate: %v", err)
	}

	if err := svc.validateCapability(&dto.AppDBResolveReq{
		Capability:  capability,
		User:        "alice",
		App:         "crm",
		Version:     "v1",
		PackagePath: "sales/accounts",
	}); err == nil {
		t.Fatal("different package should be rejected")
	}
}

func TestAppDatabaseRouterCapabilityRejectsMigrationAccess(t *testing.T) {
	svc := newTestAppDatabaseService(t)
	capability, err := svc.IssueCapability("alice", "crm", "v1", "/sales/leads/list")
	if err != nil {
		t.Fatalf("IssueCapability: %v", err)
	}

	err = svc.validateCapability(&dto.AppDBResolveReq{
		Capability:  capability,
		User:        "alice",
		App:         "crm",
		Version:     "v1",
		PackagePath: "sales/leads",
		Access:      dto.AppDBAccessMigration,
	})
	if err == nil {
		t.Fatal("router-scoped capability should not resolve migration access")
	}
	if !strings.Contains(err.Error(), "lifecycle capability") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAppDatabaseUpdateCapabilityAllowsLifecycleMigration(t *testing.T) {
	svc := newTestAppDatabaseService(t)
	capability, err := svc.IssueCapability("alice", "crm", "v1", "")
	if err != nil {
		t.Fatalf("IssueCapability: %v", err)
	}

	if err := svc.validateCapability(&dto.AppDBResolveReq{
		Capability:  capability,
		User:        "alice",
		App:         "crm",
		Version:     "v1",
		PackagePath: "sales/leads",
		Access:      dto.AppDBAccessMigration,
	}); err != nil {
		t.Fatalf("update capability should validate package migration: %v", err)
	}
}

func TestNormalizeAppDBAccess(t *testing.T) {
	tests := map[string]string{
		"":                       dto.AppDBAccessRuntime,
		"runtime":                dto.AppDBAccessRuntime,
		" RUNTIME ":              dto.AppDBAccessRuntime,
		dto.AppDBAccessMigration: dto.AppDBAccessMigration,
	}
	for input, want := range tests {
		got, err := normalizeAppDBAccess(input)
		if err != nil {
			t.Fatalf("normalizeAppDBAccess(%q): %v", input, err)
		}
		if got != want {
			t.Fatalf("normalizeAppDBAccess(%q) = %q, want %q", input, got, want)
		}
	}
	if _, err := normalizeAppDBAccess("admin"); err == nil {
		t.Fatal("unsupported access should fail")
	}
}

func TestNormalizeAppDBPackagePath(t *testing.T) {
	tests := map[string]string{
		"":               appDBRootPackage,
		"/":              appDBRootPackage,
		appDBRootPackage: appDBRootPackage,
		"/sales/leads/":  "sales/leads",
	}
	for input, want := range tests {
		got, err := normalizeAppDBPackagePath(input)
		if err != nil {
			t.Fatalf("normalizeAppDBPackagePath(%q): %v", input, err)
		}
		if got != want {
			t.Fatalf("normalizeAppDBPackagePath(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestNormalizeAppDBPackagePathRejectsUnsafePaths(t *testing.T) {
	for _, input := range []string{
		" sales/leads",
		"sales/leads ",
		"sales//leads",
		"sales/./leads",
		"sales/../admin",
		`sales\leads`,
	} {
		if _, err := normalizeAppDBPackagePath(input); err == nil {
			t.Fatalf("normalizeAppDBPackagePath(%q) should fail", input)
		}
	}
}

func TestPackagePathFromRouter(t *testing.T) {
	tests := map[string]string{
		"":                            appDBRootPackage,
		"/list":                       appDBRootPackage,
		"list":                        appDBRootPackage,
		"/sales/leads/list":           "sales/leads",
		"/sales/leads/list-by-status": "sales/leads",
	}
	for input, want := range tests {
		got, err := packagePathFromRouter(input)
		if err != nil {
			t.Fatalf("packagePathFromRouter(%q): %v", input, err)
		}
		if got != want {
			t.Fatalf("packagePathFromRouter(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestPackagePathFromRouterRejectsUnsafePaths(t *testing.T) {
	for _, input := range []string{
		" /sales/leads/list",
		"/sales/leads/list ",
		"/sales//leads/list",
		"/sales/./leads/list",
		"/sales/../admin/list",
		`sales\leads\list`,
	} {
		if _, err := packagePathFromRouter(input); err == nil {
			t.Fatalf("packagePathFromRouter(%q) should fail", input)
		}
	}
}

func TestAppDatabaseCapabilityRejectsUnsafePackagePath(t *testing.T) {
	svc := newTestAppDatabaseService(t)
	capability, err := svc.IssueCapability("alice", "crm", "v1", "/sales/leads/list")
	if err != nil {
		t.Fatalf("IssueCapability: %v", err)
	}

	err = svc.validateCapability(&dto.AppDBResolveReq{
		Capability:  capability,
		User:        "alice",
		App:         "crm",
		Version:     "v1",
		PackagePath: "sales/../admin",
		Access:      dto.AppDBAccessRuntime,
	})
	if err == nil {
		t.Fatal("unsafe package path should be rejected")
	}
}
