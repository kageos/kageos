package service

import (
	"path"
	"regexp"
	"strings"
)

// The model can repair application source, not runtime infrastructure. Only
// recognized application diagnostics cross this boundary; raw errors stay in logs.
var workspaceCompilerLine = regexp.MustCompile(`(?:^|\s)((?:\./|\.\./)*[A-Za-z0-9_./-]+\.go):(\d+(?::\d+)?):\s*(.+)$`)
var workspacePhysicalReference = regexp.MustCompile(`(?:[A-Za-z]:)?/?(?:[A-Za-z0-9_.-]+/)+(?:[A-Za-z0-9_.-]+)`)

func workspaceBuildPublicFailure(workspacePath, raw string) (buildWorkspaceResultData, string) {
	var lines []string
	for _, line := range strings.Split(raw, "\n") {
		if match := workspaceCompilerLine.FindStringSubmatch(line); len(match) > 0 {
			if source := workspacePublicSourcePath(workspacePath, match[1]); source != "" {
				lines = append(lines, source+":"+match[2]+": "+workspacePublicDiagnosticText(match[3]))
			}
			continue
		}
		// Schema diagnostics refer to the public SDK and business function, not the
		// runtime that happened to report them. Discard the startup/error wrapper.
		if start := strings.Index(line, "router /"); start >= 0 {
			line = line[start:]
			if strings.Contains(line, "code/") || strings.Contains(line, "namespace/") {
				continue
			}
			lines = append(lines, workspacePublicDiagnosticText(line))
		} else if strings.HasPrefix(strings.TrimSpace(line), "field ") {
			lines = append(lines, workspacePublicDiagnosticText(strings.TrimSpace(line)))
		}
	}
	if len(lines) > 0 {
		public := strings.Join(lines, "\n")
		return buildWorkspaceFailureResult(workspacePath, public), enrichWorkspaceBuildError(public, workspacePath)
	}
	result := buildWorkspaceResultData{
		Kind: "agent_platform_build_failure", Status: "error", WorkspacePath: workspacePath,
		ErrorCode: "platform_build_failed",
		Error:     "platform_build_failed：平台构建服务失败，未确认新版本可用。诊断信息已记录。",
	}
	result.User, result.App = splitWorkspacePath(workspacePath)
	return result, result.Error
}

func workspacePublicSourcePath(workspacePath, source string) string {
	// Absolute and module paths must identify this application. Relative compiler
	// paths are emitted from its code/cmd/app build directory or module root.
	marker := "namespace/" + strings.Trim(workspacePath, "/") + "/code/api/"
	var relative string
	if i := strings.Index(source, marker); i >= 0 {
		relative = source[i+len(marker):]
	} else {
		for _, prefix := range []string{"../../api/", "../api/", "./code/api/", "code/api/"} {
			if strings.HasPrefix(source, prefix) {
				relative = strings.TrimPrefix(source, prefix)
				break
			}
		}
	}
	if relative == "" {
		return ""
	}
	clean := path.Clean(relative)
	if clean != relative || strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, "/") {
		return ""
	}
	return strings.TrimRight(workspacePath, "/") + "/" + clean
}

func workspacePublicDiagnosticText(message string) string {
	return workspacePhysicalReference.ReplaceAllStringFunc(message, func(ref string) string {
		// SDK import names and business function paths are public; host paths and
		// generated build structure are not useful instructions for the model.
		if strings.HasPrefix(ref, "github.com/kageos/kageos-sdk/") {
			return ref
		}
		if strings.HasSuffix(ref, ".table") || strings.HasSuffix(ref, ".form") || strings.HasSuffix(ref, ".chart") {
			return ref
		}
		return "[内部引用]"
	})
}
