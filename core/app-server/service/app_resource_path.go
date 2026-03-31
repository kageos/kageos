package service

import (
	"fmt"

	permissionpkg "github.com/ai-agent-os/ai-agent-os/pkg/permission"
)

func resolveUserAppFromResourcePath(resourcePath, requestedUser, requestedApp string) (string, string, error) {
	user, app := requestedUser, requestedApp
	if resourcePath != "" {
		_, parsedUser, parsedApp := permissionpkg.ParseFullCodePath(resourcePath)
		if parsedUser == "" || parsedApp == "" {
			return "", "", fmt.Errorf("resource_path 格式错误，无法解析 user 和 app: %s", resourcePath)
		}
		if user != "" && user != parsedUser {
			return "", "", fmt.Errorf("user 与 resource_path 不匹配: user=%s, resource_path=%s", user, resourcePath)
		}
		if app != "" && app != parsedApp {
			return "", "", fmt.Errorf("app 与 resource_path 不匹配: app=%s, resource_path=%s", app, resourcePath)
		}
		user = parsedUser
		app = parsedApp
	}

	if user == "" || app == "" {
		return "", "", fmt.Errorf("必须提供 resource_path 或 user/app 参数")
	}
	return user, app, nil
}
