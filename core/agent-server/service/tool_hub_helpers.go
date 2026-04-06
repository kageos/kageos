package service

import "strings"

// parseHubSourceFromPath 从目录路径解析 source_user、source_app、source_directory_path。路径格式：/user/app 或 /user/app/xxx
func parseHubSourceFromPath(dirPath string) (sourceUser, sourceApp, sourceDirectoryPath string, errMsg string) {
	dirPath = strings.TrimSpace(dirPath)
	if dirPath == "" {
		return "", "", "", "目录路径不能为空"
	}
	if !strings.HasPrefix(dirPath, "/") {
		dirPath = "/" + dirPath
	}
	trimmed := strings.TrimPrefix(dirPath, "/")
	if trimmed == "" {
		return "", "", "", "目录路径至少需要 user/app 两段，如 /user/app"
	}
	parts := strings.SplitN(trimmed, "/", 3)
	if len(parts) < 2 {
		return "", "", "", "目录路径至少需要 user/app 两段，如 /user/app"
	}
	sourceUser = strings.TrimSpace(parts[0])
	sourceApp = strings.TrimSpace(parts[1])
	if sourceUser == "" || sourceApp == "" {
		return "", "", "", "目录路径中 user 和 app 不能为空"
	}
	sourceDirectoryPath = "/" + trimmed
	return sourceUser, sourceApp, sourceDirectoryPath, ""
}
