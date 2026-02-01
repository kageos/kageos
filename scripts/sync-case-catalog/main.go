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
	apiRoot           = "namespace/luobei/demos/code/api"
	builtinDir        = "core/agent-server/prompt/content/builtin/doc/case_catalog"
	catalogPath       = "core/agent-server/prompt/content/doc/文档目录.json"
	systemPromptPath  = "core/agent-server/prompt/content/mode/dev/system_prompt.md"
	sdkEntryName      = "agent-app SDK使用手册"
	sdkPath           = "/builtin/doc/sdk/agent-app-sdk-readme"
	sdkWhenToUse      = "当你需要生成系统/应用/代码时，请先调用 read_doc(directory: \"/builtin/doc/sdk/agent-app-sdk-readme\") 获取本文档，再按文档规范动手。"
	caseCatalogBegin   = "<!-- BEGIN CASE CATALOG -->"
	caseCatalogEnd     = "<!-- END CASE CATALOG -->"
	dirTreeBegin       = "<!-- BEGIN DIRECTORY TREE -->"
	dirTreeEnd         = "<!-- END DIRECTORY TREE -->"
	treeRootLabel      = "/builtin/doc/case_catalog/"
)

// topLevelDirComment 顶级目录在树中的注释（可选）
var topLevelDirComment = map[string]string{
	"form":            "# 单 Form 包，RouterGroup: /form",
	"form_table_chart": "# Form + Table + Chart",
	"formandtable":    "# Form + Table",
	"table":           "# 单 Table",
	"tables":          "# 多 Table",
}

type DocCatalogEntry struct {
	Name         string `json:"name"`
	FullCodePath string `json:"full_code_path"`
	WhenToUse    string `json:"when_to_use"`
}

// caseInfo 用于生成 system_prompt 中的「案例按类型归类」段落
type caseInfo struct {
	Rel        string // 相对路径，如 form/excelorcsv，与目录对齐
	Name       string // 案例名（含类型）
	Category   string // 类型，如 单 Table
	ModuleDesc string // 模块说明，如「本案例有四个模块，分别是...」
	WhenToUse  string // 适合参考 一句
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

	// 根据摘要生成「可用文档」段落并写回 system_prompt.md
	section := buildCaseCatalogSection(caseInfos)
	if err := patchSystemPrompt(repoRoot, caseCatalogBegin, caseCatalogEnd, section); err != nil {
		fmt.Fprintf(os.Stderr, "patch system_prompt case catalog: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("system_prompt.md case catalog section updated\n")

	// 根据示例项目目录生成「参考项目目录结构」树并写回 system_prompt.md
	treeContent := buildDirectoryTree(apiRootAbs)
	if err := patchSystemPrompt(repoRoot, dirTreeBegin, dirTreeEnd, "```\n"+treeContent+"\n```"); err != nil {
		fmt.Fprintf(os.Stderr, "patch system_prompt directory tree: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("system_prompt.md directory tree updated\n")
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
			name, moduleDesc, whenToUse, category := parseSummary(summaryContent)
			// 目录与示例项目对齐，文档名用 prd.md：builtin/case_catalog/form/excelorcsv/prd.md
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
			whenToUseCat := buildWhenToUse(moduleDesc, whenToUse, fullPath)
			catalog = append(catalog, DocCatalogEntry{Name: name, FullCodePath: fullPath, WhenToUse: whenToUseCat})
			infos = append(infos, caseInfo{Rel: relSlash, Name: name, Category: category, ModuleDesc: moduleDesc, WhenToUse: whenToUse})
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

func parseSummary(b []byte) (name, moduleDesc, whenToUse, category string) {
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	afterTitle := false
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
			afterTitle = true
			continue
		}
		if idx := strings.Index(line, "适合参考："); idx >= 0 {
			whenToUse = strings.TrimSpace(line[idx+len("适合参考："):])
			if whenToUse != "" && !strings.HasSuffix(whenToUse, "。") {
				whenToUse += "。"
			}
			break
		}
		if afterTitle && moduleDesc == "" {
			moduleDesc = line
		}
	}
	if name == "" {
		name = "案例"
	}
	if !strings.HasPrefix(name, "案例：") {
		name = "案例：" + name
	}
	return name, moduleDesc, whenToUse, category
}

func buildWhenToUse(moduleDesc, whenToUse, fullPath string) string {
	var parts []string
	if moduleDesc != "" {
		if !strings.HasSuffix(moduleDesc, "。") {
			moduleDesc += "。"
		}
		parts = append(parts, moduleDesc)
	}
	if whenToUse != "" {
		parts = append(parts, whenToUse)
	}
	if len(parts) == 0 {
		return "需要参考本案例时，可 read_doc(directory: \"" + fullPath + "\") 获取 PRD。"
	}
	s := strings.Join(parts, " ")
	if !strings.Contains(s, "read_doc") {
		s += " 可 read_doc(directory: \"" + fullPath + "\") 获取本案例 PRD。"
	}
	return s
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

func buildCaseCatalogSection(infos []caseInfo) string {
	byCat := make(map[string][]caseInfo)
	for _, c := range infos {
		byCat[c.Category] = append(byCat[c.Category], c)
	}
	var b strings.Builder
	for _, co := range categoryOrder {
		cases := byCat[co.Key]
		if len(cases) == 0 {
			continue
		}
		b.WriteString(co.Title)
		b.WriteString("\n")
		for _, c := range cases {
			docPath := "/builtin/doc/case_catalog/" + c.Rel
			b.WriteString("- **")
			b.WriteString(c.Name)
			b.WriteString("**（read_doc 路径 `")
			b.WriteString(docPath)
			b.WriteString("`）：")
			desc := c.ModuleDesc
			if desc != "" {
				if !strings.HasSuffix(desc, "。") {
					desc += "。"
				}
				b.WriteString(desc)
				if c.WhenToUse != "" {
					b.WriteString(" ")
					b.WriteString(c.WhenToUse)
				}
			} else if c.WhenToUse != "" {
				b.WriteString(c.WhenToUse)
			} else {
				b.WriteString("可 read_doc 获取本案例 PRD 与代码。")
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func patchSystemPrompt(repoRoot, beginMarker, endMarker, section string) error {
	path := filepath.Join(repoRoot, systemPromptPath)
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(raw)
	begin := strings.Index(content, beginMarker)
	end := strings.Index(content, endMarker)
	if begin == -1 || end == -1 || end <= begin {
		return fmt.Errorf("system_prompt.md: 未找到 %s / %s", beginMarker, endMarker)
	}
	afterBegin := begin + len(beginMarker)
	newContent := content[:afterBegin] + "\n" + section + "\n" + content[end:]
	return os.WriteFile(path, []byte(newContent), 0644)
}

// buildDirectoryTree 遍历示例项目目录，生成树形文本（仅含 .go 与子目录，排除 prd.md/summary.md）
func buildDirectoryTree(root string) string {
	var b strings.Builder
	b.WriteString(treeRootLabel)
	b.WriteByte('\n')
	entries := sortedDirEntries(root)
	for i, e := range entries {
		writeTreeEntry(&b, root, "", e, i == len(entries)-1, true)
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func sortedDirEntries(dir string) []os.DirEntry {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var dirs, files []os.DirEntry
	for _, e := range entries {
		if e.Name() == "prd.md" || e.Name() == "summary.md" {
			continue
		}
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		if e.IsDir() {
			dirs = append(dirs, e)
		} else if strings.HasSuffix(e.Name(), ".go") {
			files = append(files, e)
		}
	}
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name() < dirs[j].Name() })
	// 文件：init_.go 排最前，其余按名字
	sort.Slice(files, func(i, j int) bool {
		if files[i].Name() == "init_.go" {
			return true
		}
		if files[j].Name() == "init_.go" {
			return false
		}
		return files[i].Name() < files[j].Name()
	})
	// 与手写树风格一致：先文件（如 init_.go）再子目录
	var out []os.DirEntry
	out = append(out, files...)
	out = append(out, dirs...)
	return out
}

func writeTreeEntry(b *strings.Builder, root, prefix string, e os.DirEntry, isLast bool, topLevel bool) {
	conn := "├── "
	if isLast {
		conn = "└── "
	}
	name := e.Name()
	comment := ""
	if topLevel && e.IsDir() {
		comment = topLevelDirComment[name]
	}
	b.WriteString(prefix)
	b.WriteString(conn)
	b.WriteString(name)
	if e.IsDir() {
		b.WriteString("/")
	}
	if comment != "" {
		b.WriteString(strings.Repeat(" ", max(0, 26-len(name))))
		b.WriteString(comment)
	}
	b.WriteByte('\n')
	if !e.IsDir() {
		return
	}
	subPath := filepath.Join(root, name)
	subEntries := sortedDirEntries(subPath)
	subPrefix := prefix
	if isLast {
		subPrefix += "    "
	} else {
		subPrefix += "│   "
	}
	for i, sub := range subEntries {
		writeTreeEntry(b, subPath, subPrefix, sub, i == len(subEntries)-1, false)
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
