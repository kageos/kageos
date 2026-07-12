package server

import (
	"net/http"
	pathpkg "path"
	"strconv"
	"strings"
)

func isAllowedAgentDelegatedAPI(method, requestPath string) bool {
	if !isCanonicalDelegatedPath(requestPath) {
		return false
	}
	method = strings.ToUpper(strings.TrimSpace(method))

	exact := map[string]map[string]struct{}{
		http.MethodGet: {
			"/workspace/api/v1/workspace/context":             {},
			"/workspace/api/v1/service_tree/detail":           {},
			"/workspace/api/v1/service_tree/search_functions": {},
			"/workspace/api/v1/service_tree/search_resources": {},
			"/workspace/api/v1/team_access/my_permissions":    {},
		},
		http.MethodPost: {
			"/workspace/api/v1/service_tree/add_functions": {},
			"/workspace/api/v1/docs/crud":                  {},
			"/workspace/api/v1/packages":                   {},
			"/workspace/api/v1/workspace/files/write":      {},
			"/workspace/api/v1/workspace/files/replace":    {},
			"/workspace/api/v1/workspace/files/delete":     {},
			"/workspace/api/v1/workspace/logs/read":        {},
			"/workspace/api/v1/app/update":                 {},
			"/hr/api/v1/users":                             {},
			"/storage/api/v1/files/resolve":                {},
		},
	}
	if _, ok := exact[method][requestPath]; ok {
		return true
	}

	switch method {
	case http.MethodGet:
		if hasDelegatedPathRemainder(requestPath, "/workspace/api/v1/docs/info") ||
			hasDelegatedPathRemainder(requestPath, "/workspace/api/v1/table/search") ||
			hasDelegatedPathRemainder(requestPath, "/workspace/api/v1/chart/query") {
			return true
		}
		for _, resourceType := range []string{"form", "table", "chart"} {
			if hasDelegatedPathRemainder(requestPath, "/workspace/api/v1/function/info/"+resourceType) {
				return true
			}
		}
	case http.MethodPost:
		for _, prefix := range []string{
			"/workspace/api/v1/form/submit",
			"/workspace/api/v1/runtime/python",
			"/workspace/api/v1/table/create",
			"/workspace/api/v1/callback/on_select_fuzzy",
		} {
			if hasDelegatedPathRemainder(requestPath, prefix) {
				return true
			}
		}
	case http.MethodPut:
		if hasPositiveNumericPathRemainder(requestPath, "/workspace/api/v1/docs/crud") ||
			hasDelegatedPathRemainder(requestPath, "/workspace/api/v1/table/update") {
			return true
		}
	case http.MethodDelete:
		return hasDelegatedPathRemainder(requestPath, "/workspace/api/v1/table/delete")
	}
	return false
}

func isAllowedAgentDelegatedTimer(method, requestPath string) bool {
	if !isCanonicalDelegatedPath(requestPath) {
		return false
	}
	const tasksPath = "/timer/api/v1/tasks"
	method = strings.ToUpper(strings.TrimSpace(method))
	if requestPath == tasksPath {
		return method == http.MethodGet || method == http.MethodPost
	}
	if !strings.HasPrefix(requestPath, tasksPath+"/") {
		return false
	}
	segments := strings.Split(strings.TrimPrefix(requestPath, tasksPath+"/"), "/")
	if len(segments) == 0 || !isPositiveInteger(segments[0]) {
		return false
	}
	switch len(segments) {
	case 1:
		return method == http.MethodGet || method == http.MethodPut || method == http.MethodDelete
	case 2:
		if method == http.MethodGet && segments[1] == "executions" {
			return true
		}
		if method != http.MethodPost {
			return false
		}
		switch segments[1] {
		case "pause", "resume", "cancel", "run_now":
			return true
		}
	case 3:
		return method == http.MethodGet && segments[1] == "executions" && isPositiveInteger(segments[2])
	}
	return false
}

func isCanonicalDelegatedPath(requestPath string) bool {
	return strings.HasPrefix(requestPath, "/") &&
		!strings.Contains(requestPath, "//") &&
		pathpkg.Clean(requestPath) == requestPath
}

func hasDelegatedPathRemainder(requestPath, prefix string) bool {
	return strings.HasPrefix(requestPath, strings.TrimRight(prefix, "/")+"/")
}

func hasPositiveNumericPathRemainder(requestPath, prefix string) bool {
	if !hasDelegatedPathRemainder(requestPath, prefix) {
		return false
	}
	remainder := strings.TrimPrefix(requestPath, strings.TrimRight(prefix, "/")+"/")
	return !strings.Contains(remainder, "/") && isPositiveInteger(remainder)
}

func isPositiveInteger(value string) bool {
	id, err := strconv.ParseInt(value, 10, 64)
	return err == nil && id > 0
}
