// sync-case-catalog 从示例项目（namespace/luobei/demos/code/api）同步 prd.md + 业务 .go 代码 → system/prompt/case_catalog/*/prd.md，summary.md → readme.md。
// 用法：在项目根目录执行 go run ./scripts/sync-case-catalog
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	apiRoot    = "namespace/luobei/demos/code/api"
	builtinDir = "core/agent-server/prompt/system/prompt/case_catalog"
)

// caseInfo 表示一个已同步的案例目录。
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

	if err := validateSyncInputs(apiRootAbs); err != nil {
		fmt.Fprintf(os.Stderr, "validate inputs: %v\n", err)
		os.Exit(1)
	}

	cleanOldFlatCaseDocs(builtinAbs)

	caseInfos, err := collectCaseEntries(apiRootAbs, builtinAbs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "collect cases: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("sync ok: %d case entries\n", len(caseInfos))
}

func validateSyncInputs(apiRootAbs string) error {
	info, err := os.Stat(apiRootAbs)
	if err != nil {
		return fmt.Errorf("%s not found; restore the demos source directory before running sync-case-catalog", apiRoot)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", apiRoot)
	}

	return nil
}

// cleanOldFlatCaseDocs 删除 system/prompt/case_catalog 下旧的扁平 .md（如 form_excelorcsv.md），只保留与目录对齐的子路径文档
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

func collectCaseEntries(apiRootAbs, builtinAbs string) ([]caseInfo, error) {
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
			readmeDest := filepath.Join(destDir, "readme.md")
			if err := os.MkdirAll(destDir, 0755); err != nil {
				return err
			}
			_ = os.Remove(filepath.Join(builtinAbs, rel) + ".md")
			if err := os.WriteFile(dest, docContent, 0644); err != nil {
				return err
			}
			if err := os.WriteFile(readmeDest, summaryContent, 0644); err != nil {
				return err
			}
			infos = append(infos, caseInfo{Rel: relSlash, Name: name, Category: category, ModuleDesc: moduleDesc, KeyFeatures: keyFeatures})
			fmt.Printf("  %s -> %s/prd.md (%s)\n", rel, relSlash, name)
		}
		return nil
	})
	return infos, err
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
