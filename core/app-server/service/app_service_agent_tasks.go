package service

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/pkg/scheduledsdk"
)

const (
	ScheduledAgentSessionExecutorKey = "agent.session"
	agentTaskPolicyCreateIfMissing   = "create_if_missing"
)

func (a *AppService) reconcilePackageAgentTasks(ctx context.Context, state *appMetadataSyncState, packages []*dto.PackageInfo) error {
	if len(packages) == 0 {
		return nil
	}
	client := newAppScheduleClient()
	managedCtx := contextx.WithClientSource(ctx, scheduledTaskSourceManifest)
	for _, pkg := range packages {
		if pkg == nil || len(pkg.AgentTasks) == 0 {
			continue
		}
		existing, err := listManifestAgentTasksForPackage(managedCtx, client, pkg.FullPath)
		if err != nil {
			return fmt.Errorf("查询默认 Agent 任务 %s 失败: %w", pkg.FullPath, err)
		}
		for _, taskConfig := range pkg.AgentTasks {
			policy, err := normalizePackageAgentTaskPolicy(pkg.FullPath, taskConfig)
			if err != nil {
				return err
			}
			current := existing[strings.TrimSpace(taskConfig.Code)]
			req, err := buildPackageAgentTaskRequest(ctx, state, pkg, taskConfig)
			if err != nil {
				return err
			}
			if current != nil && policy == agentTaskPolicyCreateIfMissing {
				if _, err := client.UpdateTask(managedCtx, current.ID, updateTaskRequestFromCreate(req)); err != nil {
					return fmt.Errorf("更新默认 Agent 任务 %s/%s 失败: %w", pkg.FullPath, taskConfig.Code, err)
				}
				logger.Infof(ctx, "[PackageAgentTask] updated manifest task definition full_code_path=%s code=%s task_id=%d",
					pkg.FullPath, taskConfig.Code, current.ID)
				continue
			}
			task, err := client.CreateTask(managedCtx, req)
			if err != nil {
				return fmt.Errorf("创建默认 Agent 任务 %s/%s 失败: %w", pkg.FullPath, taskConfig.Code, err)
			}
			if task == nil || task.ID <= 0 {
				return fmt.Errorf("创建默认 Agent 任务 %s/%s 未返回有效 task_id", pkg.FullPath, taskConfig.Code)
			}
		}
	}
	return nil
}

func listManifestAgentTasksForPackage(ctx context.Context, client appScheduleClient, fullCodePath string) (map[string]*scheduledsdk.Task, error) {
	fullCodePath = strings.TrimSpace(fullCodePath)
	resp, err := client.ListTasks(ctx, scheduledsdk.ListTasksRequest{
		ExecutorKey:   ScheduledAgentSessionExecutorKey,
		Category:      "scheduled_agent_session",
		SourceType:    "agent_session",
		SourceRef:     fullCodePath,
		ResourceScope: "workspace_directory",
		ResourceKey:   fullCodePath,
		PageSize:      100,
	})
	if err != nil {
		return nil, err
	}
	out := make(map[string]*scheduledsdk.Task)
	if resp == nil {
		return out, nil
	}
	for _, task := range resp.List {
		if task == nil || task.Metadata == nil {
			continue
		}
		if strings.TrimSpace(task.Metadata["managed_by"]) != "app_manifest" {
			continue
		}
		code := strings.TrimSpace(task.Metadata["schedule_code"])
		if code == "" {
			continue
		}
		out[code] = task
	}
	return out, nil
}

func normalizePackageAgentTaskPolicy(fullCodePath string, task dto.AgentTaskConfig) (string, error) {
	policy := strings.TrimSpace(task.Policy)
	if policy == "" {
		policy = agentTaskPolicyCreateIfMissing
	}
	if policy != agentTaskPolicyCreateIfMissing {
		code := strings.TrimSpace(task.Code)
		if code == "" {
			code = "<empty>"
		}
		return "", fmt.Errorf("默认 Agent 任务 %s/%s policy 不支持: %s", strings.TrimSpace(fullCodePath), code, policy)
	}
	return policy, nil
}

func buildPackageAgentTaskRequest(ctx context.Context, state *appMetadataSyncState, pkg *dto.PackageInfo, task dto.AgentTaskConfig) (scheduledsdk.CreateTaskRequest, error) {
	if pkg == nil {
		return scheduledsdk.CreateTaskRequest{}, fmt.Errorf("默认 Agent 任务缺少 package")
	}
	fullCodePath := strings.TrimSpace(pkg.FullPath)
	if fullCodePath == "" {
		return scheduledsdk.CreateTaskRequest{}, fmt.Errorf("默认 Agent 任务 %q 缺少目录路径", task.Code)
	}
	code := strings.TrimSpace(task.Code)
	if code == "" {
		return scheduledsdk.CreateTaskRequest{}, fmt.Errorf("默认 Agent 任务 %s 缺少 code", fullCodePath)
	}
	message := strings.TrimSpace(task.Message)
	if message == "" {
		return scheduledsdk.CreateTaskRequest{}, fmt.Errorf("默认 Agent 任务 %s/%s 缺少 message", fullCodePath, code)
	}
	schedule, err := scheduledSDKScheduleFromAgentTask(ctx, fullCodePath, code, task)
	if err != nil {
		return scheduledsdk.CreateTaskRequest{}, fmt.Errorf("默认 Agent 任务 %s/%s 计划错误: %w", fullCodePath, code, err)
	}
	modeCode := strings.TrimSpace(task.ModeCode)
	if modeCode == "" {
		modeCode = "dev"
	}
	executorPayload := map[string]interface{}{
		"full_code_path":  fullCodePath,
		"message":         message,
		"display_content": message,
	}
	if modeCode != "" && modeCode != "dev" {
		executorPayload["mode_code"] = modeCode
	}
	if files := strings.TrimSpace(task.Files); files != "" {
		executorPayload["files"] = files
	}
	if task.LLMConfigID > 0 {
		executorPayload["llm_config_id"] = task.LLMConfigID
	}
	if task.MaxDurationSeconds > 0 {
		executorPayload["max_duration_seconds"] = task.MaxDurationSeconds
	}
	title := strings.TrimSpace(task.Title)
	if title == "" {
		title = strings.TrimSpace(pkg.Name)
	}
	if title == "" {
		title = "默认 Agent 任务"
	}
	requestUser := requestUserForPackageAgentTask(ctx, state)
	status := scheduledsdk.TaskStatusPaused
	if task.Enabled {
		status = scheduledsdk.TaskStatusPending
	}
	return scheduledsdk.CreateTaskRequest{
		Title:           title,
		Description:     strings.TrimSpace(task.Description),
		Category:        "scheduled_agent_session",
		Tags:            []string{"agent", "session", "app_manifest"},
		IdempotencyKey:  packageAgentTaskIdempotencyKey(fullCodePath, code),
		ExecutorKey:     ScheduledAgentSessionExecutorKey,
		ExecutorPayload: mustRawJSON(executorPayload),
		Metadata: map[string]string{
			"kind":            "scheduled_agent_session",
			"managed_by":      "app_manifest",
			"origin":          scheduledTaskOriginManifest,
			"default_enabled": fmt.Sprintf("%t", task.Enabled),
			"schedule_code":   code,
			"mode_code":       modeCode,
		},
		Status:          status,
		Schedule:        schedule,
		SourceType:      "agent_session",
		SourceRef:       fullCodePath,
		ResourceScope:   "workspace_directory",
		ResourceKey:     fullCodePath,
		RequestUser:     requestUser,
		RequestUserDept: contextx.GetRequestDepartmentFullPath(ctx),
		CreatedBy:       requestUser,
	}, nil
}

func scheduledSDKScheduleFromAgentTask(ctx context.Context, fullCodePath string, code string, task dto.AgentTaskConfig) (scheduledsdk.Schedule, error) {
	cronExpr := strings.TrimSpace(task.CronExpr)
	hasCron := cronExpr != ""
	hasEvery := task.EverySeconds > 0
	if hasCron == hasEvery {
		return scheduledsdk.Schedule{}, fmt.Errorf("必须且只能设置 cron_expr 或 every_seconds")
	}
	schedule := scheduledsdk.Schedule{
		MaxRuns: task.MaxRuns,
	}
	if hasCron {
		schedule.Type = scheduledsdk.ScheduleCron
		schedule.CronExpr = cronExpr
		schedule.Timezone = agentTaskTimezone(ctx, fullCodePath, code, task.Timezone)
		return schedule, schedule.Validate()
	}
	schedule.Type = scheduledsdk.ScheduleEvery
	schedule.IntervalSeconds = task.EverySeconds
	return schedule, schedule.Validate()
}

func agentTaskTimezone(ctx context.Context, fullCodePath string, code string, timezone string) string {
	timezone = strings.TrimSpace(timezone)
	if timezone == "" {
		return ""
	}
	if _, err := time.LoadLocation(timezone); err != nil {
		logger.Warnf(ctx, "[PackageAgentTask] invalid timezone, fallback to runtime local timezone: full_code_path=%s code=%s timezone=%q error=%v", fullCodePath, code, timezone, err)
		return ""
	}
	return timezone
}

func requestUserForPackageAgentTask(ctx context.Context, state *appMetadataSyncState) string {
	requestUser := strings.TrimSpace(contextx.GetRequestUser(ctx))
	if requestUser == "" && state != nil && state.requestUser != "" {
		requestUser = state.requestUser
	}
	if requestUser == "" && state != nil && state.app != nil {
		requestUser = state.app.User
	}
	if requestUser == "" {
		requestUser = "system"
	}
	return requestUser
}

func packageAgentTaskIdempotencyKey(fullCodePath string, code string) string {
	parts := strings.Join([]string{strings.TrimSpace(fullCodePath), strings.TrimSpace(code)}, "\x00")
	sum := sha1.Sum([]byte(parts))
	return "app-agent-task-" + hex.EncodeToString(sum[:])
}
