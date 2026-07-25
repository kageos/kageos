package builder

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kageos/kageos/pkg/buildtrace"
	"github.com/kageos/kageos/pkg/sdkmodule"
)

func TestGenerateVersionUsesProvidedReleasesDir(t *testing.T) {
	t.Parallel()

	workDir := t.TempDir()
	legacyReleasesDir := filepath.Join(workDir, "namespace", "alice", "demo", "workplace", "bin", "releases")
	if err := os.MkdirAll(legacyReleasesDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacyReleasesDir, "alice_demo_v9"), []byte("legacy"), 0644); err != nil {
		t.Fatal(err)
	}

	releasesDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(releasesDir, "alice_demo_v2"), []byte("current"), 0644); err != nil {
		t.Fatal(err)
	}

	got := NewBuilder(workDir).generateVersion("alice", "demo", releasesDir)
	if got != "v3" {
		t.Fatalf("generateVersion() = %q, want v3", got)
	}
}

func TestGoModHasSDKReplace(t *testing.T) {
	t.Parallel()

	goMod := []byte(`module demo

go 1.25.0

replace github.com/kageos/kageos-sdk => ../../../../kageos-sdk

require github.com/kageos/kageos-sdk v0.0.0
`)

	if !goModHasSDKReplace("go.mod", goMod) {
		t.Fatalf("goModHasSDKReplace() = false, want true for %s replace", sdkmodule.ModulePath)
	}
}

func TestGoModHasSDKReplaceFalseWithoutReplace(t *testing.T) {
	t.Parallel()

	goMod := []byte(`module demo

go 1.25.0

require github.com/kageos/kageos-sdk v0.2.1
`)

	if goModHasSDKReplace("go.mod", goMod) {
		t.Fatal("goModHasSDKReplace() = true, want false without replace")
	}
}

func TestBuildDependencyPreparationPlanUsesCache(t *testing.T) {
	now := time.Now()
	state := &dependencyCacheState{
		FormatVersion: dependencyCacheFormat,
		Fingerprint:   "same",
		SDKCheckedAt:  now.Add(-time.Hour),
	}

	if got := buildDependencyPreparationPlan(state, "same", true, now); got != (dependencyPreparationPlan{}) {
		t.Fatalf("local replace cache hit plan = %#v, want no work", got)
	}
	if got := buildDependencyPreparationPlan(state, "changed", true, now); !got.RunTidy || got.SyncSDK {
		t.Fatalf("changed local replace plan = %#v, want tidy only", got)
	}
	if got := buildDependencyPreparationPlan(state, "same", false, now.Add(sdkLatestCheckInterval)); !got.RunTidy || !got.SyncSDK {
		t.Fatalf("expired remote SDK plan = %#v, want SDK sync and tidy", got)
	}
}

func TestDependencyInputFingerprintIgnoresGoBodyChanges(t *testing.T) {
	moduleRoot := t.TempDir()
	writeTestFile(t, filepath.Join(moduleRoot, "go.mod"), "module example.com/demo\n\ngo 1.25.0\n")
	writeTestFile(t, filepath.Join(moduleRoot, "go.sum"), "")
	sourceFile := filepath.Join(moduleRoot, "code", "main.go")
	writeTestFile(t, sourceFile, "package main\n\nimport \"fmt\"\n\nfunc main() { fmt.Println(\"one\") }\n")

	first, _, err := dependencyInputFingerprint(moduleRoot)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, sourceFile, "package main\n\nimport \"fmt\"\n\nfunc main() { fmt.Println(\"two\") }\n")
	second, _, err := dependencyInputFingerprint(moduleRoot)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatal("function body-only change should keep dependency fingerprint stable")
	}

	writeTestFile(t, sourceFile, "package main\n\nimport (\n\t\"fmt\"\n\t\"strings\"\n)\n\nfunc main() { fmt.Println(strings.TrimSpace(\"two\")) }\n")
	third, _, err := dependencyInputFingerprint(moduleRoot)
	if err != nil {
		t.Fatal(err)
	}
	if second == third {
		t.Fatal("import change should invalidate dependency fingerprint")
	}
}

func TestDependencyCacheStateRoundTrip(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "cache", "dependencies.json")
	want := &dependencyCacheState{
		FormatVersion: dependencyCacheFormat,
		Fingerprint:   "fingerprint",
		SDKCheckedAt:  time.Now().UTC().Truncate(time.Second),
	}
	if err := saveDependencyCacheState(filename, want); err != nil {
		t.Fatal(err)
	}
	got, err := loadDependencyCacheState(filename)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.Fingerprint != want.Fingerprint || !got.SDKCheckedAt.Equal(want.SDKCheckedAt) {
		t.Fatalf("cache state = %#v, want %#v", got, want)
	}
}

func TestBuildLdFlagsCanStripReleaseDebugSymbols(t *testing.T) {
	builder := NewBuilder(t.TempDir())

	stripped := builder.buildLdFlags(&BuildOpts{
		User:              "alice",
		App:               "demo",
		Version:           "v2",
		StripDebugSymbols: true,
	})
	if !containsString(stripped, "-s") || !containsString(stripped, "-w") {
		t.Fatalf("stripped ldflags = %#v, want -s and -w", stripped)
	}

	debuggable := builder.buildLdFlags(&BuildOpts{
		User:    "alice",
		App:     "demo",
		Version: "v2",
	})
	if containsString(debuggable, "-s") || containsString(debuggable, "-w") {
		t.Fatalf("debuggable ldflags = %#v, did not expect -s or -w", debuggable)
	}
}

func TestPrepareDependenciesTracesFingerprintAndCacheLookup(t *testing.T) {
	moduleRoot := t.TempDir()
	writeTestFile(t, filepath.Join(moduleRoot, "go.mod"), `module example.com/demo

go 1.25.0

require github.com/kageos/kageos-sdk v0.0.0

replace github.com/kageos/kageos-sdk => ../../kageos-sdk
`)
	writeTestFile(t, filepath.Join(moduleRoot, "go.sum"), "")
	writeTestFile(t, filepath.Join(moduleRoot, "code", "main.go"), "package main\n")

	fingerprint, _, err := dependencyInputFingerprint(moduleRoot)
	if err != nil {
		t.Fatal(err)
	}
	cacheFile := filepath.Join(moduleRoot, filepath.FromSlash(dependencyCacheDirectory), dependencyCacheFileName)
	if err := saveDependencyCacheState(cacheFile, &dependencyCacheState{
		FormatVersion: dependencyCacheFormat,
		Fingerprint:   fingerprint,
	}); err != nil {
		t.Fatal(err)
	}

	ctx, trace := buildtrace.Ensure(context.Background(), "builder.test", "alice", "demo")
	if err := NewBuilder(moduleRoot).prepareDependencies(ctx, moduleRoot); err != nil {
		t.Fatal(err)
	}

	gotNames := make(map[string]bool)
	for _, span := range trace.Snapshot().Spans {
		gotNames[span.Name] = true
	}
	for _, want := range []string{
		"builder.dependency_fingerprint",
		"builder.dependency_cache_lookup",
		"builder.prepare_dependencies",
	} {
		if !gotNames[want] {
			t.Fatalf("missing build trace span %q in %#v", want, gotNames)
		}
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func writeTestFile(t *testing.T, filename, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
