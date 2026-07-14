package repository

import "strings"

// ResolveMessageWorkspacePath returns the canonical resource that owns the
// message conversation. The parent path is only a fallback for legacy messages
// that did not record a concrete source/full path; agent-server resolves its
// separate execution directory when it starts the workspace session.
func ResolveMessageWorkspacePath(sourcePath, fullCodePath, parentPath, _ string) string {
	for _, candidate := range []string{sourcePath, fullCodePath} {
		candidate = normalizeMessageWorkspacePath(candidate)
		if candidate == "" {
			continue
		}
		return candidate
	}
	return normalizeMessageWorkspacePath(parentPath)
}

func normalizeMessageWorkspacePath(value string) string {
	value = strings.TrimSpace(value)
	if idx := strings.IndexAny(value, "?#"); idx >= 0 {
		value = value[:idx]
	}
	return strings.TrimRight(strings.TrimSpace(value), "/")
}
