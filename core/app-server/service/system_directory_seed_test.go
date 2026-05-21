package service

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	appmodel "github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/core/app-server/repository"
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
	for _, rel := range []string{"system/tools/z.json", "system/tools/a.json", "system/tools/readme.md", "system/openapi/platform.json"} {
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
	want := []string{"system/openapi/platform.json", "system/tools/a.json", "system/tools/z.json"}
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
		{name: "system openapi root", targetPath: "/system/openapi", want: "openapi"},
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

func TestSystemDirectorySeedShouldInstallOnlyBeforeFirstVersion(t *testing.T) {
	if !systemDirectorySeedShouldInstall("") {
		t.Fatal("empty initial app version should install")
	}
	if !systemDirectorySeedShouldInstall("   ") {
		t.Fatal("blank initial app version should install")
	}
	if systemDirectorySeedShouldInstall("v1") {
		t.Fatal("non-empty initial app version should skip")
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
	if err := appRepo.CreateApp(&appmodel.App{User: SystemUsername, Code: "openapi", Name: "平台接口"}); err != nil {
		t.Fatalf("create openapi app: %v", err)
	}

	serviceTreeService := &ServiceTreeService{
		capabilityBundle: &serviceTreeCapabilityBundleService{appRepo: appRepo},
	}
	versions, err := initialSystemDirectorySeedAppVersions(serviceTreeService, []systemDirectorySeedFile{
		{appCode: "tools"},
		{appCode: "tools"},
		{appCode: "openapi"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if versions["tools"] != "v7" {
		t.Fatalf("tools version = %q, want v7", versions["tools"])
	}
	if versions["openapi"] != "" {
		t.Fatalf("openapi version = %q, want empty", versions["openapi"])
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
	}
}
