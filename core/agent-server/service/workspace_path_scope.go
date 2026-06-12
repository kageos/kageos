package service

import (
	"path"
	"regexp"
	"strings"
)

var workspaceAbsolutePathRe = regexp.MustCompile(`/[A-Za-z0-9_][A-Za-z0-9_./-]*`)

func normalizeWorkspacePath(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.Trim(raw, "`'\"，,。；;：:）)】]}>")
	if raw == "" {
		return ""
	}
	raw = strings.Split(raw, "?")[0]
	if !strings.HasPrefix(raw, "/") {
		raw = "/" + raw
	}
	raw = path.Clean(raw)
	if raw == "." || raw == "/" {
		return ""
	}
	return raw
}

func workspacePathParts(fullCodePath string) []string {
	fullCodePath = strings.Trim(normalizeWorkspacePath(fullCodePath), "/")
	if fullCodePath == "" {
		return nil
	}
	return strings.Split(fullCodePath, "/")
}

func workspaceRootPath(fullCodePath string) string {
	parts := workspacePathParts(fullCodePath)
	if len(parts) < 2 {
		return normalizeWorkspacePath(fullCodePath)
	}
	return "/" + strings.Join(parts[:2], "/")
}

func workspacePathHasPrefix(fullCodePath, prefix string) bool {
	fullCodePath = normalizeWorkspacePath(fullCodePath)
	prefix = normalizeWorkspacePath(prefix)
	if fullCodePath == "" || prefix == "" {
		return false
	}
	return fullCodePath == prefix || strings.HasPrefix(fullCodePath, prefix+"/")
}

func workspacePathDirectory(fullCodePath string) string {
	fullCodePath = normalizeWorkspacePath(fullCodePath)
	if fullCodePath == "" {
		return ""
	}
	last := path.Base(fullCodePath)
	for _, suffix := range []string{".go", ".table", ".form", ".chart", ".docs", ".md", ".json"} {
		if strings.HasSuffix(last, suffix) {
			return normalizeWorkspacePath(path.Dir(fullCodePath))
		}
	}
	return fullCodePath
}

func workspaceModuleDirectory(workspaceRoot, fullCodePath string) string {
	workspaceRoot = workspaceRootPath(workspaceRoot)
	fullCodePath = workspacePathDirectory(fullCodePath)
	if workspaceRoot == "" || fullCodePath == "" || !workspacePathHasPrefix(fullCodePath, workspaceRoot) {
		return ""
	}
	rootParts := workspacePathParts(workspaceRoot)
	parts := workspacePathParts(fullCodePath)
	if len(parts) <= len(rootParts) {
		return ""
	}
	return "/" + strings.Join(parts[:len(rootParts)+1], "/")
}

func workspaceTargetDirectoryFromCandidates(baseFullCodePath string, candidates []string) string {
	base := normalizeWorkspacePath(baseFullCodePath)
	root := workspaceRootPath(base)
	if root == "" {
		return ""
	}
	for i := len(candidates) - 1; i >= 0; i-- {
		candidate := normalizeWorkspacePath(candidates[i])
		if candidate == "" || !workspacePathHasPrefix(candidate, root) {
			continue
		}
		module := workspaceModuleDirectory(root, candidate)
		if module != "" {
			return module
		}
	}
	if module := workspaceModuleDirectory(root, base); module != "" {
		return module
	}
	return ""
}

func workspacePathsFromText(text string) []string {
	matches := workspaceAbsolutePathRe.FindAllString(text, -1)
	out := make([]string, 0, len(matches))
	for _, match := range matches {
		if path := normalizeWorkspacePath(match); path != "" {
			out = appendUniqueRoleHandoffStrings(out, path)
		}
	}
	return out
}

func workspaceTargetDirectoryFromPRD(baseFullCodePath string, digest *workspaceArtifactDigest) string {
	if digest == nil || strings.TrimSpace(digest.ProjectCode) == "" {
		return ""
	}
	base := workspacePathDirectory(baseFullCodePath)
	code := strings.Trim(strings.TrimSpace(digest.ProjectCode), "/")
	if base == "" || code == "" {
		return ""
	}
	if path.Base(base) == code {
		return base
	}
	return normalizeWorkspacePath(base + "/" + code)
}

func workspaceScopedSearchDirectory(rawDirectory, currentFullCodePath string) string {
	if dir := normalizeWorkspacePath(rawDirectory); dir != "" {
		return dir
	}
	current := normalizeWorkspacePath(currentFullCodePath)
	if module := workspaceModuleDirectory(workspaceRootPath(current), current); module != "" {
		return module
	}
	return ""
}

func workspaceDirectorySearchKeyword(directory string) string {
	parts := workspacePathParts(directory)
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func workspaceCreateDirectoryTargetPath(args map[string]interface{}, executeDirectory string) string {
	if args == nil {
		return ""
	}
	parent := firstWorkspacePathStringArg(args, "directory", "full_code_path")
	parent = normalizeWorkspacePath(firstNonEmptyString(parent, executeDirectory))
	code := strings.Trim(firstWorkspacePathStringArg(args, "code"), "/")
	if parent == "" || code == "" {
		return ""
	}
	return normalizeWorkspacePath(parent + "/" + code)
}

func firstWorkspacePathStringArg(args map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value, ok := args[key].(string); ok {
			if value = strings.TrimSpace(value); value != "" {
				return value
			}
		}
	}
	return ""
}
