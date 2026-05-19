package service

import (
	"fmt"
	"strings"
)

func resolveUserAppFromResourcePath(resourcePath string) (string, string, error) {
	if strings.TrimSpace(resourcePath) == "" {
		return "", "", fmt.Errorf("必须提供 resource_path 参数")
	}

	parts := strings.Split(strings.Trim(resourcePath, "/"), "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("resource_path 格式错误，无法解析 user 和 app: %s", resourcePath)
	}
	return parts[0], parts[1], nil
}

func resolveUserAppFromRequiredResourcePath(resourcePath string) (string, string, error) {
	return resolveUserAppFromResourcePath(resourcePath)
}
