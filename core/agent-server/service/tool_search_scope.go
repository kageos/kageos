package service

import (
	"strings"

	"github.com/ai-agent-os/ai-agent-os/pkg/permission"
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

	_, currentUser, currentApp := permission.ParseFullCodePath(currentFullCodePath)
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
