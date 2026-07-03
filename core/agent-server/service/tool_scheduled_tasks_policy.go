package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/kageos/kageos/pkg/access"
	"github.com/kageos/kageos/pkg/apicall"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/scheduledsdk"
)

func isScheduledFunctionAction(action string) bool {
	switch strings.TrimSpace(action) {
	case "execute", "table_create", "table_update", "table_delete":
		return true
	default:
		return false
	}
}

func requiresScheduledFunctionBody(fullCodePath, action string) bool {
	switch strings.TrimSpace(action) {
	case "table_create", "table_update", "table_delete":
		return true
	case "execute":
		return strings.HasSuffix(strings.TrimSpace(fullCodePath), ".form")
	default:
		return false
	}
}

func methodForScheduledFunctionAction(action string) string {
	switch strings.TrimSpace(action) {
	case "table_update":
		return "PUT"
	case "table_delete":
		return "DELETE"
	default:
		return "POST"
	}
}

func scheduledFunctionRequiredAction(fullCodePath string, action string) access.Action {
	switch strings.TrimSpace(action) {
	case "table_update":
		return access.ActionUpdate
	case "table_delete":
		return access.ActionDelete
	case "execute":
		if scheduledFunctionTemplateType(fullCodePath) == "chart" {
			return access.ActionRead
		}
		return access.ActionWrite
	default:
		return access.ActionWrite
	}
}

func scheduledFunctionTemplateType(fullCodePath string) string {
	path := strings.ToLower(strings.TrimSpace(fullCodePath))
	switch {
	case strings.Contains(path, ".form"):
		return "form"
	case strings.Contains(path, ".table"):
		return "table"
	case strings.Contains(path, ".chart"):
		return "chart"
	default:
		return ""
	}
}

func requireScheduledTaskPermission(ctx context.Context, resourcePath string, action access.Action) error {
	resourcePath = access.NormalizeResourcePath(resourcePath)
	if resourcePath == "" {
		return fmt.Errorf("resource_path is required")
	}
	resp, err := apicall.MyPermissions(ctx, resourcePath)
	if err != nil {
		return err
	}
	if resp == nil || !access.HasPermission(resp.Permissions, action) {
		return fmt.Errorf("当前用户缺少 %s 权限: %s", action, resourcePath)
	}
	return nil
}

func scheduledTaskCurrentUser(ctx context.Context) (string, error) {
	user := strings.TrimSpace(contextx.GetRequestUser(ctx))
	if user == "" {
		return "", fmt.Errorf("无法获取当前用户，不能创建或管理定时任务")
	}
	return user, nil
}

func ensureScheduledTaskOwnedByCurrentUser(ctx context.Context, task *scheduledsdk.Task) error {
	if task == nil {
		return fmt.Errorf("任务不存在")
	}
	user, err := scheduledTaskCurrentUser(ctx)
	if err != nil {
		return err
	}
	if task.CreatedBy == user || task.RequestUser == user {
		return nil
	}
	return fmt.Errorf("只能管理当前用户创建或代执行的定时任务")
}

func normalizeScheduledTaskKind(raw string) string {
	switch strings.TrimSpace(raw) {
	case "function", "agent_session":
		return strings.TrimSpace(raw)
	default:
		return "all"
	}
}

func scheduledExecutorKeyForKind(kind string) string {
	switch kind {
	case "function":
		return "app.function"
	case "agent_session":
		return "agent.session"
	default:
		return ""
	}
}

func scheduledResourceScopeForKind(kind string) string {
	switch kind {
	case "function":
		return "function"
	case "agent_session":
		return "workspace_directory"
	default:
		return ""
	}
}

func defaultScheduledFunctionTitle(fullCodePath string) string {
	name := strings.Trim(strings.TrimSpace(fullCodePath), "/")
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		name = name[idx+1:]
	}
	if name == "" {
		name = "函数"
	}
	return name + " 定时任务"
}

func defaultScheduledAgentTitle(fullCodePath string, message string) string {
	message = strings.TrimSpace(message)
	if message != "" {
		runes := []rune(message)
		if len(runes) > 18 {
			return string(runes[:18]) + "..."
		}
		return message
	}
	name := strings.Trim(strings.TrimSpace(fullCodePath), "/")
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		name = name[idx+1:]
	}
	if name == "" {
		name = "工作台"
	}
	return name + " Agent 任务"
}

func defaultScheduledAgentTitleFromMessage(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return ""
	}
	lines := strings.Split(message, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(strings.Trim(line, "#*- 　\t"))
		if line == "" {
			continue
		}
		runes := []rune(line)
		if len(runes) > 40 {
			return string(runes[:40]) + "..."
		}
		return line
	}
	runes := []rune(message)
	if len(runes) > 40 {
		return string(runes[:40]) + "..."
	}
	return message
}

func scheduledManageActionLabel(action string) string {
	switch strings.TrimSpace(action) {
	case "pause":
		return "暂停"
	case "resume":
		return "开启"
	case "cancel":
		return "取消"
	case "delete":
		return "删除"
	default:
		return "更新"
	}
}
