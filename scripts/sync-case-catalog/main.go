// sync-case-catalog 从示例项目（namespace/luobei/demos/code/api）同步 prd.md + 业务 .go 代码 → builtin/case_catalog/*.md，summary.md → 文档目录.json。
// 用法：在项目根目录执行 go run ./scripts/sync-case-catalog
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	apiRoot              = "namespace/luobei/demos/code/api"
	builtinDir           = "core/agent-server/prompt/content/builtin/doc/case_catalog"
	catalogPath          = "core/agent-server/prompt/content/doc/文档目录.json"
	createProjectDocPath = "core/agent-server/prompt/content/builtin/doc/workspace/create-project/01-create-project.md"
	sdkEntryName         = "agent-app SDK使用手册"
	sdkPath              = "/builtin/doc/sdk/agent-app-sdk-readme"
	sdkWhenToUse         = "生成系统/应用/代码前必读。框架用法：结构体标签、Table/Form/Chart 模式、注册方式、目录约定。"
	caseCatalogBegin     = "<!-- BEGIN CASE CATALOG -->"
	caseCatalogEnd       = "<!-- END CASE CATALOG -->"
)

type DocCatalogEntry struct {
	Name         string `json:"name"`
	FullCodePath string `json:"full_code_path"`
	WhenToUse    string `json:"when_to_use"`
}

// caseInfo 用于生成 create-project 中的案例索引表和文档目录.json
type caseInfo struct {
	Rel         string // 相对路径，如 form/excelorcsv，与目录对齐
	Name        string // 案例名（含类型）
	Category    string // 类型，如 单 Table
	ModuleDesc  string // 模块说明（**模块**：后的内容）
	KeyFeatures string // 关键特性标签（**关键特性**：后的内容）
}

func main() {
	repoRoot, err := findRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "find repo root: %v\n", err)
		os.Exit(1)
	}
	apiRootAbs := filepath.Join(repoRoot, apiRoot)
	builtinAbs := filepath.Join(repoRoot, builtinDir)
	catalogAbs := filepath.Join(repoRoot, catalogPath)

	cleanOldFlatCaseDocs(builtinAbs)

	entries, caseInfos, err := collectCaseEntries(repoRoot, apiRootAbs, builtinAbs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "collect cases: %v\n", err)
		os.Exit(1)
	}

	// 保留 SDK 条目，案例条目用本次同步结果
	catalog := []DocCatalogEntry{
		{Name: sdkEntryName, FullCodePath: sdkPath, WhenToUse: sdkWhenToUse},
	}
	catalog = append(catalog, entries...)

	if err := writeCatalog(catalogAbs, catalog); err != nil {
		fmt.Fprintf(os.Stderr, "write catalog: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("sync ok: %d case entries, catalog written to %s\n", len(entries), catalogPath)

	// 根据摘要生成「案例按类型归类」段落并写回 create-project/01-create-project.md
	section := buildCaseCatalogSection(caseInfos)
	if err := patchDoc(repoRoot, createProjectDocPath, caseCatalogBegin, caseCatalogEnd, section); err != nil {
		fmt.Fprintf(os.Stderr, "patch create-project case catalog: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("01-create-project.md case catalog section updated\n")
}

// cleanOldFlatCaseDocs 删除 builtin/case_catalog 下旧的扁平 .md（如 form_excelorcsv.md），只保留与目录对齐的子路径文档
func cleanOldFlatCaseDocs(builtinAbs string) {
	entries, err := os.ReadDir(builtinAbs)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(e.Name(), ".md") {
			_ = os.Remove(filepath.Join(builtinAbs, e.Name()))
		}
	}
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found")
		}
		dir = parent
	}
}

func collectCaseEntries(repoRoot, apiRootAbs, builtinAbs string) ([]DocCatalogEntry, []caseInfo, error) {
	var catalog []DocCatalogEntry
	var infos []caseInfo
	err := filepath.Walk(apiRootAbs, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() {
			return err
		}
		prd := filepath.Join(path, "prd.md")
		summary := filepath.Join(path, "summary.md")
		if readFile(prd) == nil && readFile(summary) == nil {
			rel, _ := filepath.Rel(apiRootAbs, path)
			if rel == "." {
				return nil
			}
			relSlash := filepath.ToSlash(rel)
			prdContent, err := os.ReadFile(prd)
			if err != nil {
				return err
			}
			summaryContent, err := os.ReadFile(summary)
			if err != nil {
				return err
			}
			name, moduleDesc, keyFeatures, category := parseSummary(summaryContent)
			docContent := appendCaseCode(path, prdContent)
			destDir := filepath.Join(builtinAbs, rel)
			dest := filepath.Join(destDir, "prd.md")
			if err := os.MkdirAll(destDir, 0755); err != nil {
				return err
			}
			_ = os.Remove(filepath.Join(builtinAbs, rel) + ".md")
			if err := os.WriteFile(dest, docContent, 0644); err != nil {
				return err
			}
			fullPath := "/builtin/doc/case_catalog/" + relSlash
			whenToUseCat := buildWhenToUse(keyFeatures, fullPath)
			catalog = append(catalog, DocCatalogEntry{Name: name, FullCodePath: fullPath, WhenToUse: whenToUseCat})
			infos = append(infos, caseInfo{Rel: relSlash, Name: name, Category: category, ModuleDesc: moduleDesc, KeyFeatures: keyFeatures})
			fmt.Printf("  %s -> %s/prd.md (%s)\n", rel, relSlash, name)
		}
		return nil
	})
	return catalog, infos, err
}

func readFile(path string) error {
	_, err := os.Stat(path)
	return err
}

// appendCaseCode 在 PRD 内容后追加该案例目录下业务 .go 代码（排除 init_.go，系统自动生成无需参考）
func appendCaseCode(caseDir string, prdContent []byte) []byte {
	entries, err := os.ReadDir(caseDir)
	if err != nil {
		return prdContent
	}
	var goFiles []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(e.Name(), ".go") && e.Name() != "init_.go" {
			goFiles = append(goFiles, e.Name())
		}
	}
	if len(goFiles) == 0 {
		return prdContent
	}
	sort.Strings(goFiles)
	var b strings.Builder
	b.Write(prdContent)
	b.WriteString("\n\n---\n\n## 代码实现\n\n")
	b.WriteString("以下为本案目录下 Go 源码，供 read_doc 时一并参考。\n\n")
	for _, name := range goFiles {
		content, err := os.ReadFile(filepath.Join(caseDir, name))
		if err != nil {
			continue
		}
		b.WriteString("### ")
		b.WriteString(name)
		b.WriteString("\n\n```go\n")
		b.Write(content)
		if len(content) > 0 && content[len(content)-1] != '\n' {
			b.WriteByte('\n')
		}
		b.WriteString("```\n\n")
	}
	return []byte(b.String())
}

func parseSummary(b []byte) (name, moduleDesc, keyFeatures, category string) {
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "# ") {
			name = strings.TrimPrefix(line, "# ")
			if i := strings.Index(name, "（"); i >= 0 {
				if j := strings.Index(name[i:], "）"); j >= 0 {
					category = name[i+len("（") : i+j]
				}
			}
			continue
		}
		if strings.HasPrefix(line, "**模块**：") {
			moduleDesc = strings.TrimPrefix(line, "**模块**：")
			continue
		}
		if strings.HasPrefix(line, "**关键特性**：") {
			keyFeatures = strings.TrimPrefix(line, "**关键特性**：")
			continue
		}
		// 兼容旧格式
		if idx := strings.Index(line, "适合参考："); idx >= 0 {
			if keyFeatures == "" {
				keyFeatures = strings.TrimSpace(line[idx+len("适合参考："):])
			}
		}
	}
	if name == "" {
		name = "案例"
	}
	if !strings.HasPrefix(name, "案例：") {
		name = "案例：" + name
	}
	return name, moduleDesc, keyFeatures, category
}

func buildWhenToUse(keyFeatures, fullPath string) string {
	if keyFeatures == "" {
		return "关键特性：见 read_doc(\"" + fullPath + "\")。"
	}
	return "关键特性：" + keyFeatures
}

func writeCatalog(path string, catalog []DocCatalogEntry) error {
	data, err := json.MarshalIndent(catalog, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}

// categoryOrder 用于「案例按类型归类」的段落顺序
var categoryOrder = []struct {
	Key   string
	Title string
}{
	{"单 Table", "### 1. 单 Table（仅一个 GET Table、一个 .go、纯列表 CRUD）"},
	{"单 Form", "### 2. 单 Form（仅 FormTemplate POST，无 Table）"},
	{"多 Table", "### 3. 多 Table（多个 GET Table、多 .go、主从表等，无 POST Form 或 Form 仅辅助）"},
	{"Table + Form", "### 4. Table + Form（GET Table + POST Form，无图表统计）"},
	{"Table + Form + Chart", "### 5. Table + Form + Chart（Table + Form + 统计图表）"},
}

// categoryTableLabel 类型列在表格中的显示（1. 单 Table 等）
var categoryTableLabel = map[string]string{
	"单 Table":              "1. 单 Table",
	"单 Form":               "2. 单 Form",
	"多 Table":              "3. 多 Table",
	"Table + Form":         "4. Table + Form",
	"Table + Form + Chart": "5. Table + Form + Chart",
}

// escapeTableCell 去掉单元格内换行和管道符，避免破坏 Markdown 表格
func escapeTableCell(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "|", "｜")
	return strings.TrimSpace(s)
}

func buildCaseCatalogSection(infos []caseInfo) string {
	byCat := make(map[string][]caseInfo)
	for _, c := range infos {
		byCat[c.Category] = append(byCat[c.Category], c)
	}
	var b strings.Builder
	b.WriteString("| 案例 | read_doc 路径 | 关键特性 |\n")
	b.WriteString("|------|---------------|----------|\n")
	for _, co := range categoryOrder {
		cases := byCat[co.Key]
		if len(cases) == 0 {
			continue
		}
		for _, c := range cases {
			docPath := "/builtin/doc/case_catalog/" + c.Rel
			caseName := strings.TrimPrefix(c.Name, "案例：")
			features := c.KeyFeatures
			if features == "" {
				features = "见 read_doc"
			}
			b.WriteString("| ")
			b.WriteString(escapeTableCell(caseName))
			b.WriteString(" | `")
			b.WriteString(docPath)
			b.WriteString("` | ")
			b.WriteString(escapeTableCell(features))
			b.WriteString(" |\n")
		}
	}
	return strings.TrimSuffix(b.String(), "\n")
}

// patchDoc 在指定文档中替换 beginMarker 与 endMarker 之间的内容为 section
func patchDoc(repoRoot, relPath, beginMarker, endMarker, section string) error {
	path := filepath.Join(repoRoot, relPath)
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(raw)
	begin := strings.Index(content, beginMarker)
	end := strings.Index(content, endMarker)
	if begin == -1 || end == -1 || end <= begin {
		return fmt.Errorf("%s: 未找到 %s / %s", relPath, beginMarker, endMarker)
	}
	afterBegin := begin + len(beginMarker)
	newContent := content[:afterBegin] + "\n" + section + "\n" + content[end:]
	return os.WriteFile(path, []byte(newContent), 0644)
}
