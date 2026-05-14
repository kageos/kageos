package apicall

import (
	"net/url"
	"strings"
)

func buildEscapedPath(prefix string, segments ...string) string {
	trimmedPrefix := strings.TrimRight(prefix, "/")
	if len(segments) == 0 {
		return trimmedPrefix
	}

	var builder strings.Builder
	builder.WriteString(trimmedPrefix)
	for _, segment := range segments {
		builder.WriteString("/")
		builder.WriteString(url.PathEscape(strings.Trim(segment, "/")))
	}
	return builder.String()
}

func buildWorkspaceFunctionPath(prefix, fullCodePath string) string {
	return prefix + normalizeWorkspaceFunctionPath(fullCodePath)
}

func buildWorkspaceInfoPath(prefix, fullCodePath string) string {
	normalized := strings.Trim(normalizeWorkspaceFunctionPath(fullCodePath), "/")
	if normalized == "" {
		return strings.TrimRight(prefix, "/")
	}
	return strings.TrimRight(prefix, "/") + "/" + normalized
}

func normalizeWorkspaceFunctionPath(fullCodePath string) string {
	trimmed := strings.TrimSpace(fullCodePath)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "/") {
		return trimmed
	}
	return "/" + trimmed
}
