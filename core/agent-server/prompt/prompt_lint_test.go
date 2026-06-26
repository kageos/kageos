package prompt

import (
	"encoding/json"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/kageos/kageos-sdk/agent-app/widget"
	"github.com/kageos/kageos/pkg/sdkmodule"
)

var promptRRGGBBPattern = regexp.MustCompile(`^[0-9A-Fa-f]{6}$`)
var promptWidgetTagPattern = regexp.MustCompile(`widget:"([^"]*)"`)
var promptRouteSuffixGoFilePattern = regexp.MustCompile(`\b[\w-]+\.(?:table|form|chart)\.go\b`)
var promptJSONHexFieldPattern = regexp.MustCompile(`json:"[0-9A-Fa-f]{6}"`)

func TestPromptOptionsColorsUseRRGGBB(t *testing.T) {
	walkPromptMarkdown(t, func(path, content string) {
		lines := strings.Split(content, "\n")
		for lineNo, line := range lines {
			for _, value := range extractOptionsColorsValues(line) {
				if value == "" {
					t.Fatalf("%s:%d empty options_colors value in %q", path, lineNo+1, line)
				}
				for _, color := range strings.Split(value, ",") {
					color = strings.TrimSpace(color)
					if !promptRRGGBBPattern.MatchString(color) {
						t.Fatalf("%s:%d options_colors must be RRGGBB, got %q in %q", path, lineNo+1, color, line)
					}
				}
			}
		}
	})
}

func TestPromptMarkdownDoesNotContainGenerationArtifacts(t *testing.T) {
	forbidden := []string{
		`""""options_colors`,
		`=~ s`,
		`do { my`,
		`909399:`,
		`widget 909399`,
		`options_colors 支持预设颜色`,
		`options_colors 支持预设`,
	}
	walkPromptMarkdown(t, func(path, content string) {
		for _, pattern := range forbidden {
			if strings.Contains(content, pattern) {
				t.Fatalf("%s contains forbidden prompt artifact %q", path, pattern)
			}
		}
		if match := promptJSONHexFieldPattern.FindString(content); match != "" {
			t.Fatalf("%s contains color value in json tag %q", path, match)
		}
	})
}

func TestPromptAndSeedDoNotExposeRetiredPlatformCapabilities(t *testing.T) {
	retiredSeedPatterns := []string{
		"scheduled_task",
		"scheduled_agent_task",
		"scheduled_tasks",
		"scheduled_agent_tasks",
		"message-server",
		"control-service",
		"backup-service",
		"FormOperateLog",
		"UpgradeEnterprise",
		"OnTableCreateInBatches",
		"quick_link",
		"config_management",
	}
	seedPaths := collectSeedBundlePaths(t)
	for _, seedPath := range seedPaths {
		data, err := os.ReadFile(seedPath)
		if err != nil {
			t.Fatalf("read seed bundle %s: %v", seedPath, err)
		}
		if !json.Valid(data) {
			t.Fatalf("seed bundle must be valid JSON: %s", seedPath)
		}
		content := string(data)
		for _, pattern := range retiredSeedPatterns {
			if strings.Contains(content, pattern) {
				t.Fatalf("%s exposes retired platform capability %q", seedPath, pattern)
			}
		}
	}

	forbiddenPromptPatterns := []string{
		"FormOperateLog",
		"UpgradeEnterprise",
		"OnTableCreateInBatches",
		"quick_link",
		"config_management",
		"申请链接",
		"权限不足",
		"智能体定时任务",
	}
	walkPromptMarkdown(t, func(path, content string) {
		for _, pattern := range forbiddenPromptPatterns {
			if strings.Contains(content, pattern) {
				t.Fatalf("%s exposes retired prompt capability %q", path, pattern)
			}
		}
	})
}

func collectSeedBundlePaths(t *testing.T) []string {
	t.Helper()

	seedRoot := "../../app-server/system-seed/system"
	var seedPaths []string
	if err := filepath.WalkDir(seedRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if strings.HasSuffix(entry.Name(), ".capability-bundle.json") {
			seedPaths = append(seedPaths, path)
		}
		return nil
	}); err != nil {
		t.Fatalf("walk seed bundles: %v", err)
	}
	if len(seedPaths) == 0 {
		t.Fatalf("no seed capability bundles found under %s", seedRoot)
	}
	return seedPaths
}

func TestPromptMarkdownDoesNotUseRouteSuffixAsGoFileName(t *testing.T) {
	walkPromptMarkdown(t, func(path, content string) {
		lines := strings.Split(content, "\n")
		for lineNo, line := range lines {
			if match := promptRouteSuffixGoFilePattern.FindString(line); match != "" {
				t.Fatalf("%s:%d route suffix must stay in packageContext route string, not Go file name %q in %q", path, lineNo+1, match, line)
			}
		}
	})
}

func TestPromptWidgetTagsUseSupportedTypesAndKeys(t *testing.T) {
	supportedWidgetTypes := stringSet(widget.SupportedTypes())
	walkPromptMarkdown(t, func(path, content string) {
		lines := strings.Split(content, "\n")
		for lineNo, line := range lines {
			for _, match := range promptWidgetTagPattern.FindAllStringSubmatch(line, -1) {
				tag := strings.TrimSpace(match[1])
				if tag == "-" {
					continue
				}
				for _, segment := range strings.Split(tag, ";") {
					segment = strings.TrimSpace(segment)
					if segment == "" {
						continue
					}
					key, value, ok := strings.Cut(segment, ":")
					if !ok || strings.TrimSpace(key) == "" {
						t.Fatalf("%s:%d widget tag segment must be key:value, got %q in %q", path, lineNo+1, segment, line)
					}
					key = strings.TrimSpace(key)
					value = strings.TrimSpace(value)
					if key == "type" {
						if _, ok := supportedWidgetTypes[value]; !ok {
							t.Fatalf("%s:%d unsupported widget type %q in %q", path, lineNo+1, value, line)
						}
					}
				}
				widgetType := parsedTagValue(tag, "type")
				if widgetType == "" || widgetType == "-" {
					continue
				}
				allowedKeys := stringSet(widget.AllowedTagKeys(widgetType))
				for _, segment := range strings.Split(tag, ";") {
					key, _, ok := strings.Cut(strings.TrimSpace(segment), ":")
					if !ok {
						continue
					}
					key = strings.TrimSpace(key)
					if _, ok := allowedKeys[key]; !ok {
						t.Fatalf("%s:%d unsupported widget key %q for widget %q in %q", path, lineNo+1, key, widgetType, line)
					}
				}
			}
		}
	})
}

func TestCaseCatalogDoesNotUseKnownBrokenSDKExamples(t *testing.T) {
	forbidden := []string{
		`req.GetPage()`,
		`req.GetPageSize()`,
		`.Time.Format(`,
		`StartTime.Format(`,
		`EndTime.Format(`,
	}
	walkPromptMarkdown(t, func(path, content string) {
		if !strings.Contains(path, "system/prompt/case_catalog/") {
			return
		}
		for _, pattern := range forbidden {
			if strings.Contains(content, pattern) {
				t.Fatalf("%s contains known broken SDK example %q", path, pattern)
			}
		}
	})
}

func TestCaseCatalogSDKSelectorsExist(t *testing.T) {
	exports := exportedSDKSymbols(t)
	selectorPattern := regexp.MustCompile(`\b(app|types|chart|response|callback|statistics)\.([A-Z][A-Za-z0-9_]*)\b`)

	walkPromptMarkdown(t, func(path, content string) {
		if !strings.Contains(path, "system/prompt/case_catalog/") {
			return
		}
		lines := strings.Split(content, "\n")
		for lineNo, line := range lines {
			for _, match := range selectorPattern.FindAllStringSubmatch(line, -1) {
				alias, symbol := match[1], match[2]
				if _, ok := exports[alias][symbol]; !ok {
					t.Fatalf("%s:%d uses SDK selector %s.%s but %s is not exported by SDK package %s", path, lineNo+1, alias, symbol, symbol, alias)
				}
			}
		}
	})
}

func walkPromptMarkdown(t *testing.T, visit func(path, content string)) {
	t.Helper()
	err := fs.WalkDir(promptFS, "system/prompt", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		data, err := promptFS.ReadFile(path)
		if err != nil {
			return err
		}
		visit(path, string(data))
		return nil
	})
	if err != nil {
		t.Fatalf("walk prompt markdown: %v", err)
	}
}

func exportedSDKSymbols(t *testing.T) map[string]map[string]struct{} {
	t.Helper()
	sdkRoot := findSDKRootForPromptLint(t)
	packages := map[string]string{
		"app":        "agent-app/app",
		"callback":   "agent-app/callback",
		"chart":      "agent-app/chart",
		"response":   "agent-app/response",
		"statistics": "agent-app/statistics",
		"types":      "agent-app/types",
	}

	result := make(map[string]map[string]struct{}, len(packages))
	for alias, relPath := range packages {
		result[alias] = exportedSymbolsInDir(t, filepath.Join(sdkRoot, filepath.FromSlash(relPath)))
	}
	return result
}

func findSDKRootForPromptLint(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("resolve test file for %s@%s", sdkmodule.ModulePath, sdkmodule.Version)
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "../../.."))
	sibling := filepath.Join(filepath.Dir(repoRoot), "kageos-sdk")
	if _, err := os.Stat(filepath.Join(sibling, "agent-app")); err == nil {
		return sibling
	}

	for _, root := range filepath.SplitList(build.Default.GOPATH) {
		if root == "" {
			continue
		}
		dir := filepath.Join(root, "pkg", "mod", sdkmodule.ModulePath+"@"+sdkmodule.Version)
		if _, err := os.Stat(filepath.Join(dir, "agent-app")); err == nil {
			return dir
		}
	}

	t.Fatalf("cannot find %s@%s in GOPATH module cache or sibling repo %s", sdkmodule.ModulePath, sdkmodule.Version, sibling)
	return ""
}

func exportedSymbolsInDir(t *testing.T, dir string) map[string]struct{} {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read SDK dir %s: %v", dir, err)
	}

	symbols := map[string]struct{}{}
	fset := token.NewFileSet()
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		filePath := filepath.Join(dir, entry.Name())
		file, err := parser.ParseFile(fset, filePath, nil, 0)
		if err != nil {
			t.Fatalf("parse SDK file %s: %v", filePath, err)
		}
		for _, decl := range file.Decls {
			switch decl := decl.(type) {
			case *ast.FuncDecl:
				if ast.IsExported(decl.Name.Name) {
					symbols[decl.Name.Name] = struct{}{}
				}
			case *ast.GenDecl:
				for _, spec := range decl.Specs {
					switch spec := spec.(type) {
					case *ast.TypeSpec:
						if ast.IsExported(spec.Name.Name) {
							symbols[spec.Name.Name] = struct{}{}
						}
					case *ast.ValueSpec:
						for _, name := range spec.Names {
							if ast.IsExported(name.Name) {
								symbols[name.Name] = struct{}{}
							}
						}
					}
				}
			}
		}
	}
	return symbols
}

func stringSet(items []string) map[string]struct{} {
	result := make(map[string]struct{}, len(items))
	for _, item := range items {
		result[item] = struct{}{}
	}
	return result
}

func parsedTagValue(tag string, wantedKey string) string {
	for _, segment := range strings.Split(tag, ";") {
		key, value, ok := strings.Cut(strings.TrimSpace(segment), ":")
		if !ok {
			continue
		}
		if strings.TrimSpace(key) == wantedKey {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func extractOptionsColorsValues(line string) []string {
	const marker = "options_colors:"
	var values []string
	for {
		idx := strings.Index(line, marker)
		if idx < 0 {
			return values
		}
		line = line[idx+len(marker):]
		end := len(line)
		for i, r := range line {
			if r == ';' || r == '`' || r == '"' || r == '\'' || r == ' ' || r == '\t' || r == '\r' {
				end = i
				break
			}
		}
		values = append(values, line[:end])
		line = line[end:]
	}
}
