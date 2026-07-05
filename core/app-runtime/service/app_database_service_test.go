package service

import (
	"strings"
	"testing"

	"github.com/kageos/kageos/dto"
	appconfig "github.com/kageos/kageos/pkg/config"
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
