package prompt

import (
	"context"
	"encoding/json"
	"io/fs"
	"path"
	"sort"
	"strings"
	"sync"

	workspaceprd "github.com/kageos/kageos/core/agent-server/workspace/prd"
)

const (
	SystemPromptRootPath                 = "/system/prompt"
	SystemPromptWorkspaceEnvTemplatePath = SystemPromptRootPath + "/doc/workspace-env-template"
	systemPromptSeedRoot                 = "system/prompt"
	systemPromptReadmeFileName           = "readme.md"
	retiredIntentPromptPackageCode       = "inte" + "nts"
)

type PromptSeedDoc struct {
	Name        string
	Description string
	LogicalPath string
	Content     string
	Format      string
}

type PromptSeedPackage struct {
	Code        string
	Name        string
	Description string
	LogicalPath string
}

var (
	systemPromptSeedDocsOnce sync.Once
	systemPromptSeedDocs     []PromptSeedDoc
	systemPromptSeedDocsErr  error

	systemPromptSeedPackagesOnce sync.Once
	systemPromptSeedPackages     []PromptSeedPackage
	systemPromptSeedPackagesErr  error
)

func NormalizePromptDocPath(fullCodePath string) string {
	fullCodePath = strings.TrimSpace(fullCodePath)
	if fullCodePath == "" {
		return ""
	}
	if !strings.HasPrefix(fullCodePath, "/") {
		fullCodePath = "/" + fullCodePath
	}
	switch {
	case fullCodePath == SystemPromptRootPath:
		return fullCodePath
	case strings.HasPrefix(fullCodePath, SystemPromptRootPath+"/"):
		return strings.TrimRight(fullCodePath, "/")
	default:
		return ""
	}
}

func IsLegacyWorkspacePromptPath(fullCodePath string) bool {
	fullCodePath = strings.TrimSpace(fullCodePath)
	if fullCodePath == "" {
		return false
	}
	if !strings.HasPrefix(fullCodePath, "/") {
		fullCodePath = "/" + fullCodePath
	}
	fullCodePath = strings.TrimRight(fullCodePath, "/")
	return fullCodePath == SystemPromptRootPath+"/workspace" || strings.HasPrefix(fullCodePath, SystemPromptRootPath+"/workspace/")
}

func IsRetiredIntentPromptPath(fullCodePath string) bool {
	fullCodePath = strings.TrimSpace(fullCodePath)
	if fullCodePath == "" {
		return false
	}
	if !strings.HasPrefix(fullCodePath, "/") {
		fullCodePath = "/" + fullCodePath
	}
	fullCodePath = strings.TrimRight(fullCodePath, "/")
	prefix := SystemPromptRootPath + "/" + retiredIntentPromptPackageCode
	return fullCodePath == prefix || strings.HasPrefix(fullCodePath, prefix+"/")
}

func IsPromptDocPath(fullCodePath string) bool {
	logical := NormalizePromptDocPath(fullCodePath)
	return logical != "" && !IsLegacyWorkspacePromptPath(logical) && !IsRetiredIntentPromptPath(logical)
}

func LoadPromptDocCatalog(ctx context.Context) []DocCatalogEntry {
	return GetDocCatalog()
}

func LoadWorkspaceEnvTemplate(ctx context.Context) string {
	_, content := GetPromptDocContent(ctx, SystemPromptWorkspaceEnvTemplatePath)
	if strings.TrimSpace(content) != "" {
		return content
	}
	return WorkspaceEnvTemplate
}

func ResolveModeProvider(ctx context.Context, code string) WorkspaceModePromptProvider {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil
	}
	return GetModeProvider(code)
}

func LoadModeConfig(ctx context.Context, code string) *ModeConfig {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil
	}
	return loadSeedModeConfig(code)
}

func GetPromptDocContent(ctx context.Context, fullCodePath string) (name, content string) {
	logical := NormalizePromptDocPath(fullCodePath)
	if logical == "" {
		return "", ""
	}
	if IsLegacyWorkspacePromptPath(logical) || IsRetiredIntentPromptPath(logical) {
		return "", ""
	}
	name, content = getSeedPromptDocContent(logical)
	return name, finalizePromptDocContent(logical, content)
}

func finalizePromptDocContent(logical, content string) string {
	switch NormalizePromptDocPath(logical) {
	case SystemPromptRootPath + "/mode/dev/system_prompt":
		return finalizeModeSystemPrompt(content)
	case SystemPromptRootPath + "/roles/product-manager":
		return workspaceprd.ApplyContractMarkdown(content)
	default:
		return content
	}
}

func ListSystemPromptSeedDocs() ([]PromptSeedDoc, error) {
	systemPromptSeedDocsOnce.Do(func() {
		systemPromptSeedDocs, systemPromptSeedDocsErr = buildSystemPromptSeedDocs()
	})
	if systemPromptSeedDocsErr != nil {
		return nil, systemPromptSeedDocsErr
	}
	out := make([]PromptSeedDoc, len(systemPromptSeedDocs))
	copy(out, systemPromptSeedDocs)
	return out, nil
}

func ListSystemPromptSeedPackages() ([]PromptSeedPackage, error) {
	systemPromptSeedPackagesOnce.Do(func() {
		systemPromptSeedPackages, systemPromptSeedPackagesErr = buildSystemPromptSeedPackages()
	})
	if systemPromptSeedPackagesErr != nil {
		return nil, systemPromptSeedPackagesErr
	}
	out := make([]PromptSeedPackage, len(systemPromptSeedPackages))
	copy(out, systemPromptSeedPackages)
	return out, nil
}

func buildPromptDocCatalogFromSeed() []DocCatalogEntry {
	docs, err := ListSystemPromptSeedDocs()
	if err != nil {
		return nil
	}
	packages, err := ListSystemPromptSeedPackages()
	if err != nil {
		return nil
	}

	packageByPath := make(map[string]PromptSeedPackage, len(packages))
	for _, pkg := range packages {
		packageByPath[pkg.LogicalPath] = pkg
	}

	var entries []DocCatalogEntry
	for _, doc := range docs {
		switch {
		case shouldIncludeSDKDocCatalogEntry(doc):
			entries = append(entries, DocCatalogEntry{
				Name:         doc.Name,
				FullCodePath: doc.LogicalPath,
				WhenToUse:    deriveSDKWhenToUse(doc.Description),
			})
		case shouldIncludeCaseCatalogSeedEntry(doc, packageByPath):
			pkg := packageByPath[doc.LogicalPath]
			entries = append(entries, DocCatalogEntry{
				Name:         pkg.Name,
				FullCodePath: pkg.LogicalPath,
				WhenToUse:    deriveCaseCatalogWhenToUse(pkg.Description),
			})
		}
	}
	sortPromptDocCatalogEntries(entries)
	return entries
}

func getSeedPromptDocContent(logicalPath string) (name, content string) {
	docs, err := ListSystemPromptSeedDocs()
	if err != nil {
		return "", ""
	}
	logicalPath = NormalizePromptDocPath(logicalPath)
	for _, doc := range docs {
		if doc.LogicalPath == logicalPath {
			return doc.Name, doc.Content
		}
	}
	return "", ""
}

func buildSystemPromptSeedDocs() ([]PromptSeedDoc, error) {
	var docs []PromptSeedDoc

	if err := appendPromptSeedReadmeDocs(&docs); err != nil {
		return nil, err
	}

	for _, rel := range []string{
		"platform-capability-boundaries.md",
		"platform-introduction.md",
		"platform-usage-and-philosophy.md",
	} {
		logical := SystemPromptRootPath + "/" + strings.TrimSuffix(rel, path.Ext(rel))
		if err := appendPromptSeedFileDoc(&docs, systemPromptSeedRoot+"/"+rel, logical, "markdown", "", ""); err != nil {
			return nil, err
		}
	}

	for _, rel := range []struct {
		FileName string
		Format   string
		Name     string
		Desc     string
	}{
		{FileName: "workspace-env-template.md", Format: "markdown"},
	} {
		logical := SystemPromptRootPath + "/doc/" + strings.TrimSuffix(rel.FileName, path.Ext(rel.FileName))
		if err := appendPromptSeedFileDoc(&docs, systemPromptSeedRoot+"/doc/"+rel.FileName, logical, rel.Format, rel.Name, rel.Desc); err != nil {
			return nil, err
		}
	}

	if err := appendPromptRoleSeedDocs(&docs); err != nil {
		return nil, err
	}
	if err := appendPromptSDKSeedDocs(&docs); err != nil {
		return nil, err
	}
	if err := appendPromptCaseCatalogSeedDocs(&docs); err != nil {
		return nil, err
	}
	if err := appendPromptModeSeedDocs(&docs); err != nil {
		return nil, err
	}

	sort.Slice(docs, func(i, j int) bool {
		return docs[i].LogicalPath < docs[j].LogicalPath
	})
	return docs, nil
}

func buildSystemPromptSeedPackages() ([]PromptSeedPackage, error) {
	var packages []PromptSeedPackage
	err := fs.WalkDir(promptFS, systemPromptSeedRoot, func(current string, entry fs.DirEntry, err error) error {
		if err != nil || !entry.IsDir() {
			return err
		}
		if isLegacyWorkspaceSeedPath(current) || isRetiredIntentSeedPath(current) {
			if current == systemPromptSeedRoot+"/workspace" {
				return fs.SkipDir
			}
			if current == systemPromptSeedRoot+"/"+retiredIntentPromptPackageCode {
				return fs.SkipDir
			}
			return nil
		}
		logical := SystemPromptRootPath
		if current != systemPromptSeedRoot {
			rel := strings.TrimPrefix(current, systemPromptSeedRoot+"/")
			logical = SystemPromptRootPath + "/" + strings.Trim(rel, "/")
		}
		title, body, _ := loadPromptSeedReadmeMeta(current)
		code := path.Base(logical)
		if logical == SystemPromptRootPath {
			code = "prompt"
		}
		if title == "" {
			title = code
		}
		packages = append(packages, PromptSeedPackage{
			Code:        code,
			Name:        title,
			Description: body,
			LogicalPath: logical,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(packages, func(i, j int) bool {
		di := promptPathDepth(packages[i].LogicalPath)
		dj := promptPathDepth(packages[j].LogicalPath)
		if di != dj {
			return di < dj
		}
		return packages[i].LogicalPath < packages[j].LogicalPath
	})
	return packages, nil
}

func appendPromptSeedReadmeDocs(docs *[]PromptSeedDoc) error {
	return fs.WalkDir(promptFS, systemPromptSeedRoot, func(current string, entry fs.DirEntry, err error) error {
		if err != nil || !entry.IsDir() {
			return err
		}
		if isLegacyWorkspaceSeedPath(current) || isRetiredIntentSeedPath(current) {
			if current == systemPromptSeedRoot+"/workspace" {
				return fs.SkipDir
			}
			if current == systemPromptSeedRoot+"/"+retiredIntentPromptPackageCode {
				return fs.SkipDir
			}
			return nil
		}
		if shouldSkipPromptReadmeIndexDoc(current) {
			return nil
		}
		title, body, content := loadPromptSeedReadmeMeta(current)
		if strings.TrimSpace(content) == "" {
			return nil
		}
		logical := logicalPromptPathFromSeedDir(current)
		*docs = append(*docs, PromptSeedDoc{
			Name:        firstNonEmpty(title, path.Base(logical)),
			Description: body,
			LogicalPath: logical,
			Content:     content,
			Format:      "markdown",
		})
		return nil
	})
}

func appendPromptSDKSeedDocs(docs *[]PromptSeedDoc) error {
	root := systemPromptSeedRoot + "/sdk"
	if err := fs.WalkDir(promptFS, root, func(current string, entry fs.DirEntry, err error) error {
		if err != nil || !entry.IsDir() {
			return err
		}
		mdNames, readErr := listDirectMarkdownFiles(current)
		if readErr != nil || len(mdNames) == 0 {
			return readErr
		}
		content, concatErr := concatMarkdownFiles(current, mdNames)
		if concatErr != nil {
			return concatErr
		}
		logical := logicalPromptPathFromSeedDir(current)
		title, body, _ := loadPromptSeedReadmeMeta(current)
		*docs = append(*docs, PromptSeedDoc{
			Name:        firstNonEmpty(title, path.Base(logical)),
			Description: body,
			LogicalPath: logical,
			Content:     content,
			Format:      "markdown",
		})
		return nil
	}); err != nil {
		return err
	}

	return fs.WalkDir(promptFS, root, func(current string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		name := entry.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".md") || strings.EqualFold(name, systemPromptReadmeFileName) {
			return nil
		}
		dir := path.Dir(current)
		logical := logicalPromptPathFromSeedDir(dir) + "/" + strings.TrimSuffix(name, path.Ext(name))
		return appendPromptSeedFileDoc(docs, current, logical, "markdown", "", "")
	})
}

func appendPromptRoleSeedDocs(docs *[]PromptSeedDoc) error {
	root := systemPromptSeedRoot + "/roles"
	return fs.WalkDir(promptFS, root, func(current string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		name := entry.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".md") || strings.EqualFold(name, systemPromptReadmeFileName) {
			return nil
		}
		dir := path.Dir(current)
		logical := logicalPromptPathFromSeedDir(dir) + "/" + strings.TrimSuffix(name, path.Ext(name))
		return appendPromptSeedFileDoc(docs, current, logical, "markdown", "", "")
	})
}

func appendPromptCaseCatalogSeedDocs(docs *[]PromptSeedDoc) error {
	root := systemPromptSeedRoot + "/case_catalog"
	return fs.WalkDir(promptFS, root, func(current string, entry fs.DirEntry, err error) error {
		if err != nil || !entry.IsDir() {
			return err
		}
		content, format, readErr := readPromptCaseCatalogDoc(current)
		if readErr != nil {
			return readErr
		}
		if strings.TrimSpace(content) == "" {
			return nil
		}
		logical := logicalPromptPathFromSeedDir(current)
		name, desc := extractMarkdownMeta(content, path.Base(logical))
		*docs = append(*docs, PromptSeedDoc{
			Name:        name,
			Description: desc,
			LogicalPath: logical,
			Content:     content,
			Format:      format,
		})
		return nil
	})
}

func readPromptCaseCatalogDoc(dir string) (string, string, error) {
	prdJSONPath := dir + "/prd.json"
	prdJSON, jsonErr := readPromptSeedFile(prdJSONPath)
	if jsonErr != nil && !isNotExistErr(jsonErr) {
		return "", "", jsonErr
	}

	prdMDPath := dir + "/prd.md"
	prdMD, mdErr := readPromptSeedFile(prdMDPath)
	if mdErr != nil && !isNotExistErr(mdErr) {
		return "", "", mdErr
	}

	if strings.TrimSpace(prdJSON) == "" {
		return prdMD, "markdown", nil
	}

	name, _ := extractMarkdownMeta(prdMD, path.Base(dir))
	var out strings.Builder
	out.WriteString("# ")
	out.WriteString(name)
	out.WriteString("\n\n## 结构化 PRD JSON\n\n```json\n")
	out.WriteString(strings.TrimSpace(prdJSON))
	out.WriteString("\n```\n")
	if notes := extractPromptCaseImplementationNotes(prdMD); notes != "" {
		out.WriteString("\n")
		out.WriteString(notes)
	}
	return out.String(), "markdown", nil
}

func extractPromptCaseImplementationNotes(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	markers := []string{"\n## 三、", "\n## 四、", "\n## 代码实现"}
	start := -1
	for _, marker := range markers {
		if idx := strings.Index(content, marker); idx >= 0 && (start < 0 || idx < start) {
			start = idx + 1
		}
	}
	if start < 0 {
		return ""
	}
	return strings.TrimSpace(content[start:])
}

func appendPromptModeSeedDocs(docs *[]PromptSeedDoc) error {
	modeRoot := systemPromptSeedRoot + "/mode"
	modeEntries, err := fs.ReadDir(promptFS, modeRoot)
	if err != nil {
		return err
	}
	for _, modeEntry := range modeEntries {
		if !modeEntry.IsDir() {
			continue
		}
		modeCode := modeEntry.Name()
		if modeCode == "" {
			continue
		}

		cfgPath := modeRoot + "/" + modeCode + "/config.json"
		cfgContent, cfgErr := readPromptSeedFile(cfgPath)
		if cfgErr != nil {
			if isNotExistErr(cfgErr) {
				continue
			}
			return cfgErr
		}
		var cfg ModeConfig
		if err := json.Unmarshal([]byte(cfgContent), &cfg); err != nil {
			return err
		}

		for _, fileName := range []string{"config.json", "system_prompt.md"} {
			filePath := modeRoot + "/" + modeCode + "/" + fileName
			content, readErr := readPromptSeedFile(filePath)
			if readErr != nil {
				if isNotExistErr(readErr) {
					continue
				}
				return readErr
			}
			format := "markdown"
			if strings.HasSuffix(fileName, ".json") {
				format = "json"
			}
			base := strings.TrimSuffix(fileName, path.Ext(fileName))
			logical := SystemPromptRootPath + "/mode/" + modeCode + "/" + base
			name, desc := resolveModeSeedDocMeta(cfg, fileName, content)
			*docs = append(*docs, PromptSeedDoc{
				Name:        name,
				Description: desc,
				LogicalPath: logical,
				Content:     content,
				Format:      format,
			})
		}
	}
	return nil
}

func appendPromptSeedFileDoc(docs *[]PromptSeedDoc, fsPath, logicalPath, format, fallbackName, fallbackDesc string) error {
	content, err := readPromptSeedFile(fsPath)
	if err != nil {
		return err
	}
	name := fallbackName
	desc := fallbackDesc
	if format == "markdown" {
		name, desc = extractMarkdownMeta(content, firstNonEmpty(fallbackName, path.Base(logicalPath)))
	} else {
		name = firstNonEmpty(name, path.Base(logicalPath))
	}
	*docs = append(*docs, PromptSeedDoc{
		Name:        name,
		Description: desc,
		LogicalPath: logicalPath,
		Content:     content,
		Format:      format,
	})
	return nil
}

func resolveModeSeedDocMeta(cfg ModeConfig, fileName, content string) (name, desc string) {
	modeName := firstNonEmpty(strings.TrimSpace(cfg.Name), "模式")
	switch fileName {
	case "config.json":
		return modeName + "配置", strings.TrimSpace(cfg.Description)
	case "system_prompt.md":
		name, desc = extractMarkdownMeta(content, modeName+"系统提示词")
		return name, desc
	default:
		return extractMarkdownMeta(content, path.Base(fileName))
	}
}

func readPromptSeedFile(fsPath string) (string, error) {
	data, err := fs.ReadFile(promptFS, fsPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func loadPromptSeedReadmeMeta(dir string) (title, body, content string) {
	readmePath := strings.TrimRight(dir, "/") + "/" + systemPromptReadmeFileName
	content, err := readPromptSeedFile(readmePath)
	if err != nil {
		return "", "", ""
	}
	title, body = extractMarkdownMeta(content, "")
	return title, body, content
}

func logicalPromptPathFromSeedDir(dir string) string {
	if dir == systemPromptSeedRoot {
		return SystemPromptRootPath
	}
	rel := strings.TrimPrefix(dir, systemPromptSeedRoot+"/")
	return SystemPromptRootPath + "/" + strings.Trim(rel, "/")
}

func shouldSkipPromptReadmeIndexDoc(dir string) bool {
	if isLegacyWorkspaceSeedPath(dir) || isRetiredIntentSeedPath(dir) {
		return true
	}
	if strings.HasPrefix(dir, systemPromptSeedRoot+"/workspace/") {
		return true
	}
	if dir == systemPromptSeedRoot+"/sdk" || strings.HasPrefix(dir, systemPromptSeedRoot+"/sdk/") {
		return true
	}
	if _, err := fs.Stat(promptFS, strings.TrimRight(dir, "/")+"/prd.md"); err == nil {
		return true
	}
	return false
}

func isLegacyWorkspaceSeedPath(seedPath string) bool {
	seedPath = strings.TrimRight(seedPath, "/")
	return seedPath == systemPromptSeedRoot+"/workspace" || strings.HasPrefix(seedPath, systemPromptSeedRoot+"/workspace/")
}

func isRetiredIntentSeedPath(seedPath string) bool {
	seedPath = strings.TrimRight(seedPath, "/")
	prefix := systemPromptSeedRoot + "/" + retiredIntentPromptPackageCode
	return seedPath == prefix || strings.HasPrefix(seedPath, prefix+"/")
}

func listDirectMarkdownFiles(dir string) ([]string, error) {
	entries, err := fs.ReadDir(promptFS, dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(strings.ToLower(name), ".md") {
			names = append(names, name)
		}
	}
	sort.Slice(names, func(i, j int) bool {
		left := strings.ToLower(names[i])
		right := strings.ToLower(names[j])
		if left == systemPromptReadmeFileName {
			return true
		}
		if right == systemPromptReadmeFileName {
			return false
		}
		return left < right
	})
	return names, nil
}

func concatMarkdownFiles(dir string, names []string) (string, error) {
	var builder strings.Builder
	for i, name := range names {
		data, err := fs.ReadFile(promptFS, dir+"/"+name)
		if err != nil {
			return "", err
		}
		if i > 0 {
			builder.WriteString("\n\n")
		}
		if strings.EqualFold(name, systemPromptReadmeFileName) {
			builder.Write(data)
			continue
		}
		builder.WriteString("## ")
		builder.WriteString(name)
		builder.WriteString("\n\n")
		builder.Write(data)
	}
	return builder.String(), nil
}

func extractMarkdownMeta(content, fallbackName string) (name, desc string) {
	content = strings.TrimSpace(content)
	if content == "" {
		return fallbackName, ""
	}
	lines := strings.Split(content, "\n")
	bodyLines := make([]string, 0, len(lines))
	foundTitle := false
	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if !foundTitle && strings.HasPrefix(line, "# ") {
			name = strings.TrimSpace(strings.TrimPrefix(line, "# "))
			foundTitle = true
			continue
		}
		bodyLines = append(bodyLines, rawLine)
	}
	if name == "" {
		name = fallbackName
	}
	desc = strings.TrimSpace(strings.Join(bodyLines, "\n"))
	return name, desc
}

func shouldIncludeSDKDocCatalogEntry(doc PromptSeedDoc) bool {
	return strings.HasPrefix(doc.LogicalPath, SystemPromptRootPath+"/sdk/") && doc.Format == "markdown"
}

func shouldIncludeCaseCatalogSeedEntry(doc PromptSeedDoc, packageByPath map[string]PromptSeedPackage) bool {
	if !strings.HasPrefix(doc.LogicalPath, SystemPromptRootPath+"/case_catalog/") {
		return false
	}
	pkg, ok := packageByPath[doc.LogicalPath]
	if !ok {
		return false
	}
	return isLeafPromptSeedPackage(pkg.LogicalPath, packageByPath)
}

func isLeafPromptSeedPackage(logicalPath string, packageByPath map[string]PromptSeedPackage) bool {
	prefix := strings.TrimRight(logicalPath, "/") + "/"
	for path := range packageByPath {
		if path != logicalPath && strings.HasPrefix(path, prefix) {
			return false
		}
	}
	return true
}

func sortPromptDocCatalogEntries(entries []DocCatalogEntry) {
	sort.Slice(entries, func(i, j int) bool {
		li := entries[i].FullCodePath
		lj := entries[j].FullCodePath
		leftSDK := strings.HasPrefix(li, SystemPromptRootPath+"/sdk/")
		rightSDK := strings.HasPrefix(lj, SystemPromptRootPath+"/sdk/")
		if leftSDK != rightSDK {
			return leftSDK
		}
		return li < lj
	})
}

func deriveSDKWhenToUse(description string) string {
	description = strings.TrimSpace(description)
	if description == "" {
		return ""
	}
	return firstParagraph(description)
}

func deriveCaseCatalogWhenToUse(description string) string {
	description = strings.TrimSpace(description)
	if description == "" {
		return ""
	}
	for _, rawLine := range strings.Split(description, "\n") {
		line := strings.TrimSpace(rawLine)
		switch {
		case strings.HasPrefix(line, "**关键特性**："):
			return "关键特性：" + strings.TrimSpace(strings.TrimPrefix(line, "**关键特性**："))
		case strings.Contains(line, "适合参考："):
			idx := strings.Index(line, "适合参考：")
			return strings.TrimSpace(line[idx+len("适合参考："):])
		}
	}
	return firstParagraph(description)
}

func firstParagraph(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	paragraphs := strings.Split(text, "\n\n")
	return strings.TrimSpace(paragraphs[0])
}

func promptPathDepth(logicalPath string) int {
	logicalPath = strings.Trim(strings.TrimSpace(logicalPath), "/")
	if logicalPath == "" {
		return 0
	}
	return len(strings.Split(logicalPath, "/"))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func isNotExistErr(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "not exist")
}
