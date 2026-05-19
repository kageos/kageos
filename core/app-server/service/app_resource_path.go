package service

import (
	"fmt"
	"strings"
)

func resolveUserAppFromResourcePath(resourcePath, requestedUser, requestedApp string) (string, string, error) {
	user, app := requestedUser, requestedApp
	if resourcePath != "" {
		parts := strings.Split(strings.Trim(resourcePath, "/"), "/")
		parsedUser, parsedApp := "", ""
		if len(parts) >= 2 {
			parsedUser = parts[0]
			parsedApp = parts[1]
		}
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

func resolveUserAppFromRequiredResourcePath(resourcePath string) (string, string, error) {
	user, app, err := resolveUserAppFromResourcePath(resourcePath, "", "")
	if err != nil && strings.TrimSpace(resourcePath) == "" {
		return "", "", fmt.Errorf("必须提供 resource_path 参数")
	}
	return user, app, err
}
