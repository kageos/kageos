package repository

import "strings"

// ResolveMessageWorkspacePath 返回消息会话归属的具体资源；父目录仅作为旧消息缺字段时的兜底。
// agent-server 会另外解析执行目录，因此这里不能擅自把函数资源改成父目录。
func ResolveMessageWorkspacePath(sourcePath, fullCodePath, parentPath, _ string) string {
	for _, candidate := range []string{sourcePath, fullCodePath} {
		if normalized := normalizeMessageWorkspacePath(candidate); normalized != "" {
			return normalized
		}
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
