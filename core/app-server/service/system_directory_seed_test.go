package service

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	appmodel "github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/core/app-server/repository"
	"github.com/kageos/kageos/dto"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSystemDirectorySeedTargetPathUsesDirectoryAsTargetNode(t *testing.T) {
	seedDir := filepath.Join(t.TempDir(), "system-seed")
	filePath := filepath.Join(seedDir, "system", "tools", "openapi", "excel.json")

	got, err := systemDirectorySeedTargetPath(seedDir, filePath)
	if err != nil {
		t.Fatal(err)
	}
	if got != "/system/tools/openapi" {
		t.Fatalf("target path = %q, want /system/tools/openapi", got)
	}
}

func TestSystemDirectorySeedTargetPathRejectsRootJSON(t *testing.T) {
	seedDir := filepath.Join(t.TempDir(), "system-seed")
	_, err := systemDirectorySeedTargetPath(seedDir, filepath.Join(seedDir, "excel.json"))
	if err == nil {
		t.Fatal("expected root json to be rejected")
	}
}

func TestListSystemDirectorySeedFilesSorted(t *testing.T) {
	seedDir := t.TempDir()
	for _, rel := range []string{"system/tools/z.json", "system/tools/a.json", "system/tools/readme.md", "system/tools/openapi/platform.json"} {
		path := filepath.Join(seedDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	files, err := listSystemDirectorySeedFiles(seedDir)
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(files))
	for _, file := range files {
		rel, err := filepath.Rel(seedDir, file)
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, filepath.ToSlash(rel))
	}
	want := []string{"system/tools/a.json", "system/tools/openapi/platform.json", "system/tools/z.json"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("files = %#v, want %#v", got, want)
	}
}

func TestResolveSystemDirectorySeedFilesIncludesAppCode(t *testing.T) {
	seedDir := filepath.Join(t.TempDir(), "system-seed")
	filePath := filepath.Join(seedDir, "system", "tools", "openapi", "excel.json")

	got, err := resolveSystemDirectorySeedFiles(seedDir, []string{filePath})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("seed files count = %d, want 1", len(got))
	}
	if got[0].filePath != filePath {
		t.Fatalf("filePath = %q, want %q", got[0].filePath, filePath)
	}
	if got[0].targetPath != "/system/tools/openapi" {
		t.Fatalf("targetPath = %q, want /system/tools/openapi", got[0].targetPath)
	}
	if got[0].appCode != "tools" {
		t.Fatalf("appCode = %q, want tools", got[0].appCode)
	}
}

func TestSystemDirectorySeedAppCodeFromTargetPath(t *testing.T) {
	tests := []struct {
		name       string
		targetPath string
		want       string
		wantErr    bool
	}{
		{name: "system tools nested", targetPath: "/system/tools/openapi", want: "tools"},
		{name: "system openapi root", targetPath: "/system/openapi", wantErr: true},
		{name: "non system user", targetPath: "/alice/tools", wantErr: true},
		{name: "unknown system app", targetPath: "/system/unknown", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := systemDirectorySeedAppCodeFromTargetPath(tt.targetPath)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("appCode = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSystemDirectorySeedShouldInstallUntilVersionAdvances(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&appmodel.App{}, &appmodel.ServiceTree{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	serviceTreeService := &ServiceTreeService{
		capabilityBundle: &serviceTreeCapabilityBundleService{
			appRepo: repository.NewAppRepository(db),
		},
	}
	seedFile := systemDirectorySeedFile{appCode: "tools", targetPath: "/system/tools"}

	got, err := systemDirectorySeedShouldInstall(serviceTreeService, seedFile, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Fatal("empty initial app version should install")
	}

	got, err = systemDirectorySeedShouldInstall(serviceTreeService, seedFile, "v1", false)
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Fatal("v1 system seed should install so partial first-boot seeds can recover")
	}

	got, err = systemDirectorySeedShouldInstall(serviceTreeService, seedFile, "v7", false)
	if err != nil {
		t.Fatal(err)
	}
	if got {
		t.Fatal("non-empty app version above v1 should skip")
	}

	got, err = systemDirectorySeedShouldInstall(serviceTreeService, seedFile, "v1", true)
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Fatal("app created in current boot should install even when CreateApp assigned v1")
	}
}

func TestInitialSystemDirectorySeedAppVersionsReadsUniqueApps(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&appmodel.App{}); err != nil {
		t.Fatalf("migrate app: %v", err)
	}

	appRepo := repository.NewAppRepository(db)
	if err := appRepo.CreateApp(&appmodel.App{User: SystemUsername, Code: "tools", Name: "官方工具", Version: "v7"}); err != nil {
		t.Fatalf("create tools app: %v", err)
	}

	serviceTreeService := &ServiceTreeService{
		capabilityBundle: &serviceTreeCapabilityBundleService{appRepo: appRepo},
	}
	versions, err := initialSystemDirectorySeedAppVersions(serviceTreeService, []systemDirectorySeedFile{
		{appCode: "tools"},
		{appCode: "tools"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if versions["tools"] != "v7" {
		t.Fatalf("tools version = %q, want v7", versions["tools"])
	}
}

func TestSystemSeedBundlesUseCapabilitySchema(t *testing.T) {
	seedDir := filepath.Join("..", "system-seed")
	files, err := listSystemDirectorySeedFiles(seedDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("expected system seed bundles")
	}

	for _, file := range files {
		if _, err := systemDirectorySeedTargetPath(seedDir, file); err != nil {
			t.Fatalf("invalid seed target for %s: %v", file, err)
		}
		bundle, err := readCapabilityBundleFile(file)
		if err != nil {
			t.Fatalf("read seed bundle %s: %v", file, err)
		}
		if err := validateCapabilityBundle(bundle); err != nil {
			t.Fatalf("invalid capability seed %s: %v", file, err)
		}
		assertSystemSeedBundlePathsAreRelativeToTarget(t, seedDir, file, bundle)
	}
}

func assertSystemSeedBundlePathsAreRelativeToTarget(t *testing.T, seedDir, file string, bundle *dto.CapabilityBundle) {
	t.Helper()

	targetPath, err := systemDirectorySeedTargetPath(seedDir, file)
	if err != nil {
		t.Fatalf("invalid seed target for %s: %v", file, err)
	}
	appCode, err := systemDirectorySeedAppCodeFromTargetPath(targetPath)
	if err != nil {
		t.Fatalf("invalid seed app for %s: %v", file, err)
	}
	prefix := strings.Trim(appCode, "/") + "/"

	for _, pkg := range bundle.Packages {
		if pkg.Path == appCode || strings.HasPrefix(pkg.Path, prefix) {
			t.Fatalf("seed bundle %s package path %q must be relative to target %s", file, pkg.Path, targetPath)
		}
	}
	for _, sourceFile := range bundle.Files {
		if sourceFile.PackagePath == appCode || strings.HasPrefix(sourceFile.PackagePath, prefix) {
			t.Fatalf("seed bundle %s file package_path %q must be relative to target %s", file, sourceFile.PackagePath, targetPath)
		}
	}
	for _, node := range bundle.TreeNodes {
		if node.RelativePath == appCode || strings.HasPrefix(node.RelativePath, prefix) {
			t.Fatalf("seed bundle %s tree node path %q must be relative to target %s", file, node.RelativePath, targetPath)
		}
	}
}
