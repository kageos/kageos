package service

import (
	"strings"
)

const (
	searchScopeSystem      = "system"
	searchScopeVisible     = "visible"
	searchScopeCurrentUser = "current_user"
	searchScopeCurrentApp  = "current_app"
)

func normalizeSearchScope(raw string, defaultScope string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case searchScopeVisible:
		return searchScopeVisible
	case searchScopeCurrentUser:
		return searchScopeCurrentUser
	case searchScopeCurrentApp:
		return searchScopeCurrentApp
	case searchScopeSystem:
		return searchScopeSystem
	default:
		if defaultScope != "" {
			return defaultScope
		}
		return searchScopeVisible
	}
}

func resolveSearchScopeUserApp(scope string, user string, app string, currentFullCodePath string, defaultScope string) (string, string, string) {
	scope = normalizeSearchScope(scope, defaultScope)
	user = strings.TrimSpace(user)
	app = strings.TrimSpace(app)
	if user != "" || app != "" {
		return user, app, scope
	}

	pathParts := strings.Split(strings.Trim(currentFullCodePath, "/"), "/")
	currentUser, currentApp := "", ""
	if len(pathParts) >= 2 {
		currentUser = pathParts[0]
		currentApp = pathParts[1]
	}
	switch scope {
	case searchScopeSystem:
		return "system", "", scope
	case searchScopeCurrentUser:
		return currentUser, "", scope
	case searchScopeCurrentApp:
		return currentUser, currentApp, scope
	default:
		return "", "", scope
	}
}
