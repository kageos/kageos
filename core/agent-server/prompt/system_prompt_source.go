package prompt

import (
	"context"
	"encoding/json"
	"io/fs"
	"path"
	"sort"
	"strings"
	"sync"

	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
)

const (
	SystemPromptRootPath                 = "/system/prompt"
	SystemPromptWorkspaceEnvTemplatePath = SystemPromptRootPath + "/doc/workspace-env-template"
	systemPromptSeedRoot                 = "system/prompt"
	systemPromptReadmeFileName           = "readme.md"
)

type PromptSeedDoc struct {
	Name        string
	Description string
	LogicalPath string
	ActualPath  string
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

func IsPromptDocPath(fullCodePath string) bool {
	return NormalizePromptDocPath(fullCodePath) != ""
}

func PromptDocLeafPath(fullCodePath string) string {
	logical := NormalizePromptDocPath(fullCodePath)
	if logical == "" {
		return ""
	}
	return logical + ".docs"
}

func PromptDocIndexPath(fullCodePath string) string {
	logical := NormalizePromptDocPath(fullCodePath)
	if logical == "" {
		return ""
	}
	return logical + "/index.docs"
}

func PromptDocCandidatePaths(fullCodePath string) []string {
	logical := NormalizePromptDocPath(fullCodePath)
	if logical == "" {
		return nil
	}
	candidates := make([]string, 0, 3)
	if actualPath := getSeedPromptDocActualPath(logical); actualPath != "" {
		candidates = append(candidates, actualPath)
	}
	for _, candidate := range []string{logical + "/index.docs", logical + ".docs"} {
		if !containsString(candidates, candidate) {
			candidates = append(candidates, candidate)
		}
	}
	return candidates
}

func LoadPromptDocCatalog(ctx context.Context) []DocCatalogEntry {
	if entries := loadPromptDocCatalogFromTree(ctx); len(entries) > 0 {
		return entries
	}
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
	if provider := loadModeProviderFromTree(ctx, code); provider != nil {
		return provider
	}
	return GetModeProvider(code)
}

func LoadModeConfig(ctx context.Context, code string) *ModeConfig {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil
	}
	if cfg := loadModeConfigFromTree(ctx, code); cfg != nil {
		return cfg
	}
	return loadSeedModeConfig(code)
}

func GetPromptDocContent(ctx context.Context, fullCodePath string) (name, content string) {
	logical := NormalizePromptDocPath(fullCodePath)
	if logical == "" {
		return "", ""
	}
	for _, actualPath := range PromptDocCandidatePaths(logical) {
		doc, err := apicall.GetDoc(ctx, actualPath)
		if err != nil || doc == nil || strings.TrimSpace(doc.Content) == "" {
			continue
		}
		docName := strings.TrimSpace(doc.Name)
		if docName == "" {
			docName = path.Base(logical)
		}
		return docName, doc.Content
	}
	return getSeedPromptDocContent(logical)
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

func loadPromptDocCatalogFromTree(ctx context.Context) []DocCatalogEntry {
	if ctx == nil {
		return nil
	}
	var entries []DocCatalogEntry
	entries = append(entries, collectSDKCatalogEntriesFromTree(ctx, SystemPromptRootPath+"/sdk")...)
	entries = append(entries, collectCaseCatalogEntriesFromTree(ctx, SystemPromptRootPath+"/case_catalog", true)...)
	if len(entries) == 0 {
		return nil
	}
	sortPromptDocCatalogEntries(entries)
	return entries
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

func collectSDKCatalogEntriesFromTree(ctx context.Context, fullCodePath string) []DocCatalogEntry {
	workspaceCtx, err := apicall.GetWorkspaceContext(ctx, fullCodePath, "")
	if err != nil || workspaceCtx == nil {
		return nil
	}
	var entries []DocCatalogEntry
	for _, child := range workspaceCtx.Children {
		switch child.Type {
		case "package":
			entries = append(entries, collectSDKCatalogEntriesFromTree(ctx, child.FullCodePath)...)
		case "docs":
			if strings.EqualFold(child.Code, "index.docs") {
				continue
			}
			entries = append(entries, DocCatalogEntry{
				Name:         child.Name,
				FullCodePath: promptDocLogicalPathFromActualPath(child.FullCodePath),
				WhenToUse:    deriveSDKWhenToUse(child.Description),
			})
		}
	}
	return entries
}

func collectCaseCatalogEntriesFromTree(ctx context.Context, fullCodePath string, isRoot bool) []DocCatalogEntry {
	workspaceCtx, err := apicall.GetWorkspaceContext(ctx, fullCodePath, "")
	if err != nil || workspaceCtx == nil {
		return nil
	}

	var (
		entries          []DocCatalogEntry
		hasPackageChilds bool
	)
	for _, child := range workspaceCtx.Children {
		if child.Type != "package" {
			continue
		}
		hasPackageChilds = true
		entries = append(entries, collectCaseCatalogEntriesFromTree(ctx, child.FullCodePath, false)...)
	}
	if !isRoot && !hasPackageChilds {
		entries = append(entries, DocCatalogEntry{
			Name:         workspaceCtx.Directory.Name,
			FullCodePath: workspaceCtx.Directory.FullCodePath,
			WhenToUse:    deriveCaseCatalogWhenToUse(workspaceCtx.Directory.Description),
		})
	}
	return entries
}

func loadModeProviderFromTree(ctx context.Context, code string) *modeProvider {
	cfg := loadModeConfigFromTree(ctx, code)
	if cfg == nil {
		return nil
	}

	systemPrompt := loadModeDocContent(ctx, code, cfg.SystemPromptFile)
	firstAssistant := loadModeDocContent(ctx, code, cfg.FirstAssistantFile)
	var operationPrompt string
	if cfg.OperationPromptFile != "" {
		operationPrompt = loadModeDocContent(ctx, code, cfg.OperationPromptFile)
	}

	toolNames := cfg.ToolNames
	if toolNames == nil {
		toolNames = []string{}
	}
	return &modeProvider{
		code:            code,
		systemPrompt:    systemPrompt,
		firstAssistant:  firstAssistant,
		operationPrompt: operationPrompt,
		toolNames:       toolNames,
	}
}

func loadModeConfigFromTree(ctx context.Context, code string) *ModeConfig {
	_, cfgContent := GetPromptDocContent(ctx, SystemPromptRootPath+"/mode/"+code+"/config")
	if strings.TrimSpace(cfgContent) == "" {
		return nil
	}
	var cfg ModeConfig
	if err := json.Unmarshal([]byte(cfgContent), &cfg); err != nil {
		return nil
	}
	return &cfg
}

func loadModeDocContent(ctx context.Context, code, fileName string) string {
	fileName = strings.TrimSpace(fileName)
	if fileName == "" {
		return ""
	}
	base := strings.TrimSuffix(path.Base(fileName), path.Ext(fileName))
	if base == "" {
		return ""
	}
	_, content := GetPromptDocContent(ctx, SystemPromptRootPath+"/mode/"+code+"/"+base)
	return content
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

func getSeedPromptDocActualPath(logicalPath string) string {
	docs, err := ListSystemPromptSeedDocs()
	if err != nil {
		return ""
	}
	logicalPath = NormalizePromptDocPath(logicalPath)
	for _, doc := range docs {
		if doc.LogicalPath == logicalPath {
			return doc.ActualPath
		}
	}
	return ""
}

func buildSystemPromptSeedDocs() ([]PromptSeedDoc, error) {
	var docs []PromptSeedDoc

	if err := appendPromptSeedReadmeDocs(&docs); err != nil {
		return nil, err
	}

	for _, rel := range []string{"platform-overview.md", "platform-cross-cutting-capabilities.md"} {
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

	if err := appendPromptSDKSeedDocs(&docs); err != nil {
		return nil, err
	}
	if err := appendPromptWorkspaceSeedDocs(&docs); err != nil {
		return nil, err
	}
	if err := appendPromptCaseCatalogSeedDocs(&docs); err != nil {
		return nil, err
	}
	if err := appendPromptModeSeedDocs(&docs); err != nil {
		return nil, err
	}

	sort.Slice(docs, func(i, j int) bool {
		return docs[i].ActualPath < docs[j].ActualPath
	})
	return docs, nil
}

func buildSystemPromptSeedPackages() ([]PromptSeedPackage, error) {
	var packages []PromptSeedPackage
	err := fs.WalkDir(promptFS, systemPromptSeedRoot, func(current string, entry fs.DirEntry, err error) error {
		if err != nil || !entry.IsDir() {
			return err
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
			ActualPath:  PromptDocIndexPath(logical),
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
			ActualPath:  PromptDocIndexPath(logical),
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

func appendPromptWorkspaceSeedDocs(docs *[]PromptSeedDoc) error {
	root := systemPromptSeedRoot + "/workspace"
	return fs.WalkDir(promptFS, root, func(current string, entry fs.DirEntry, err error) error {
		if err != nil || !entry.IsDir() || current == root {
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
			ActualPath:  PromptDocIndexPath(logical),
			Content:     content,
			Format:      "markdown",
		})
		return nil
	})
}

func appendPromptCaseCatalogSeedDocs(docs *[]PromptSeedDoc) error {
	root := systemPromptSeedRoot + "/case_catalog"
	return fs.WalkDir(promptFS, root, func(current string, entry fs.DirEntry, err error) error {
		if err != nil || !entry.IsDir() {
			return err
		}
		prdPath := current + "/prd.md"
		content, readErr := readPromptSeedFile(prdPath)
		if readErr != nil {
			if isNotExistErr(readErr) {
				return nil
			}
			return readErr
		}
		logical := logicalPromptPathFromSeedDir(current)
		name, desc := extractMarkdownMeta(content, path.Base(logical))
		*docs = append(*docs, PromptSeedDoc{
			Name:        name,
			Description: desc,
			LogicalPath: logical,
			ActualPath:  PromptDocIndexPath(logical),
			Content:     content,
			Format:      "markdown",
		})
		return nil
	})
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

		for _, fileName := range []string{"config.json", "system_prompt.md", "first_assistant.md"} {
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
				ActualPath:  PromptDocLeafPath(logical),
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
		ActualPath:  PromptDocLeafPath(logicalPath),
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
	case "first_assistant.md":
		name, desc = extractMarkdownMeta(content, modeName+"首条助手消息")
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

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
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

func promptDocLogicalPathFromActualPath(fullCodePath string) string {
	switch {
	case strings.HasSuffix(fullCodePath, "/index.docs"):
		return strings.TrimSuffix(fullCodePath, "/index.docs")
	case strings.HasSuffix(fullCodePath, ".docs"):
		return strings.TrimSuffix(fullCodePath, ".docs")
	default:
		return fullCodePath
	}
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
