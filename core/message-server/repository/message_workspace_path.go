package repository

import "strings"

// ResolveMessageWorkspacePath keeps the concrete function as the message
// source while resolving the directory that can own a workstation session.
func ResolveMessageWorkspacePath(sourcePath, fullCodePath, parentPath, templateType string) string {
	parentPath = normalizeMessageWorkspacePath(parentPath)
	for _, candidate := range []string{sourcePath, fullCodePath} {
		candidate = normalizeMessageWorkspacePath(candidate)
		if candidate == "" {
			continue
		}
		if isMessageFunctionPath(candidate, templateType) {
			if parentPath != "" {
				return parentPath
			}
			if derived := parentMessageWorkspacePath(candidate); derived != "" {
				return derived
			}
		}
		return candidate
	}
	return parentPath
}

func normalizeMessageWorkspacePath(value string) string {
	value = strings.TrimSpace(value)
	if idx := strings.IndexAny(value, "?#"); idx >= 0 {
		value = value[:idx]
	}
	return strings.TrimRight(strings.TrimSpace(value), "/")
}

func isMessageFunctionPath(value, templateType string) bool {
	lowerPath := strings.ToLower(normalizeMessageWorkspacePath(value))
	for _, suffix := range []string{".form", ".table", ".chart"} {
		if strings.HasSuffix(lowerPath, suffix) {
			return true
		}
	}
	templateType = strings.ToLower(strings.TrimSpace(templateType))
	templateType = strings.TrimSuffix(templateType, "_template")
	templateType = strings.TrimSuffix(templateType, "template")
	switch strings.TrimSpace(templateType) {
	case "form", "table", "chart":
		return true
	default:
		return false
	}
}

func parentMessageWorkspacePath(value string) string {
	value = normalizeMessageWorkspacePath(value)
	if idx := strings.LastIndex(value, "/"); idx > 0 {
		return strings.TrimRight(value[:idx], "/")
	}
	return ""
}
