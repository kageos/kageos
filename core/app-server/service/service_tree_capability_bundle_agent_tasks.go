package service

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/scheduledsdk"
)

const capabilityBundleAgentTaskPageSize = 100

type scheduledAgentSessionPayload struct {
	FullCodePath       string `json:"full_code_path"`
	Message            string `json:"message"`
	DisplayContent     string `json:"display_content"`
	ModeCode           string `json:"mode_code"`
	MaxDurationSeconds int64  `json:"max_duration_seconds"`
}

func (s *serviceTreeCapabilityBundleService) appendCapabilityBundleAgentTasks(
	ctx context.Context,
	bundle *dto.CapabilityBundle,
	baseTree *model.ServiceTree,
	rootTree *model.ServiceTree,
	includeBaseCode bool,
	seenAgentTasks map[string]struct{},
) error {
	if rootTree == nil || strings.TrimSpace(rootTree.FullCodePath) == "" {
		return nil
	}
	client := newAppScheduleClient()
	for page := 1; ; page++ {
		resp, err := client.ListTasks(ctx, scheduledsdk.ListTasksRequest{
			ExecutorKey:       ScheduledAgentSessionExecutorKey,
			Category:          "scheduled_agent_session",
			ResourceScope:     "workspace_directory",
			ResourceKeyPrefix: rootTree.FullCodePath,
			Page:              page,
			PageSize:          capabilityBundleAgentTaskPageSize,
		})
		if err != nil {
			return fmt.Errorf("查询 Agent 任务失败: %w", err)
		}
		if resp == nil || len(resp.List) == 0 {
			return nil
		}
		for _, task := range resp.List {
			item, err := capabilityBundleAgentTaskFromScheduledTask(baseTree, task, includeBaseCode)
			if err != nil {
				return err
			}
			if item == nil {
				continue
			}
			key := capabilityAgentTaskKey(item.RelativePath, item.Code)
			if _, exists := seenAgentTasks[key]; exists {
				item.Code = fmt.Sprintf("%s_%d", item.Code, task.ID)
				key = capabilityAgentTaskKey(item.RelativePath, item.Code)
			}
			seenAgentTasks[key] = struct{}{}
			bundle.AgentTasks = append(bundle.AgentTasks, item)
		}
		if len(resp.List) < capabilityBundleAgentTaskPageSize {
			return nil
		}
	}
}

func capabilityBundleAgentTaskFromScheduledTask(baseTree *model.ServiceTree, task *scheduledsdk.Task, includeBaseCode bool) (*dto.CapabilityBundleAgentTask, error) {
	if task == nil || task.ExecutorKey != ScheduledAgentSessionExecutorKey {
		return nil, nil
	}
	resourcePath := scheduledAgentTaskResourcePath(task)
	if resourcePath == "" {
		return nil, nil
	}
	node := &model.ServiceTree{
		Code:         path.Base(resourcePath),
		FullCodePath: resourcePath,
		Type:         model.ServiceTreeTypePackage,
	}
	relativePath, err := capabilityRelativePackagePath(baseTree, node, includeBaseCode)
	if err != nil {
		return nil, err
	}
	if relativePath == "" {
		return nil, nil
	}

	payload := decodeScheduledAgentSessionPayload(task.ExecutorPayload)
	message := strings.TrimSpace(payload.Message)
	if message == "" {
		message = strings.TrimSpace(payload.DisplayContent)
	}
	if message == "" {
		message = strings.TrimSpace(task.Description)
	}
	code := capabilityBundleAgentTaskCode(task)
	return &dto.CapabilityBundleAgentTask{
		RelativePath:       relativePath,
		Code:               code,
		Title:              strings.TrimSpace(task.Title),
		Description:        strings.TrimSpace(task.Description),
		Message:            message,
		Enabled:            task.Status == scheduledsdk.TaskStatusPending,
		Schedule:           task.Schedule,
		ModeCode:           firstNonEmptyString(strings.TrimSpace(payload.ModeCode), strings.TrimSpace(task.Metadata["mode_code"])),
		MaxDurationSeconds: payload.MaxDurationSeconds,
		Policy:             agentTaskPolicyCreateIfMissing,
	}, nil
}

func scheduledAgentTaskResourcePath(task *scheduledsdk.Task) string {
	if task == nil {
		return ""
	}
	if path := strings.TrimSpace(task.ResourceKey); path != "" {
		return path
	}
	if path := strings.TrimSpace(task.SourceRef); path != "" {
		return path
	}
	payload := decodeScheduledAgentSessionPayload(task.ExecutorPayload)
	return strings.TrimSpace(payload.FullCodePath)
}

func decodeScheduledAgentSessionPayload(raw json.RawMessage) scheduledAgentSessionPayload {
	var payload scheduledAgentSessionPayload
	if len(raw) == 0 {
		return payload
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return scheduledAgentSessionPayload{}
	}
	return payload
}

func capabilityBundleAgentTaskCode(task *scheduledsdk.Task) string {
	if task == nil {
		return ""
	}
	for _, key := range []string{"schedule_code", "bundle_task_code"} {
		if task.Metadata != nil {
			if code := normalizeCapabilityAgentTaskCode(task.Metadata[key]); code != "" {
				return code
			}
		}
	}
	if task.ID > 0 {
		return fmt.Sprintf("task_%d", task.ID)
	}
	return "task"
}

func normalizeCapabilityAgentTaskCode(code string) string {
	code = strings.TrimSpace(code)
	if code == "" {
		return ""
	}
	replacer := strings.NewReplacer("/", "_", "\\", "_", " ", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_")
	code = replacer.Replace(code)
	code = strings.Trim(code, "_-.")
	if code == "" {
		return ""
	}
	return code
}

func capabilityAgentTaskKey(relativePath, code string) string {
	return strings.Trim(strings.TrimSpace(relativePath), "/") + ":" + strings.TrimSpace(code)
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func (s *serviceTreeCapabilityBundleService) installCapabilityBundleAgentTasks(
	ctx context.Context,
	targetRootPath string,
	tasks []*dto.CapabilityBundleAgentTask,
) ([]string, error) {
	client := newAppScheduleClient()
	created := make([]string, 0, len(tasks))
	for _, task := range tasks {
		if task == nil {
			continue
		}
		targetFullCodePath := joinCapabilityFullCodePath(targetRootPath, task.RelativePath)
		req, err := buildCapabilityBundleAgentTaskRequest(ctx, targetFullCodePath, task)
		if err != nil {
			return created, err
		}
		createdTask, err := client.CreateTask(ctx, req)
		if err != nil {
			return created, fmt.Errorf("%s/%s: %w", targetFullCodePath, task.Code, err)
		}
		if createdTask == nil || createdTask.ID <= 0 {
			return created, fmt.Errorf("%s/%s 未返回有效 task_id", targetFullCodePath, task.Code)
		}
		created = append(created, capabilityAgentTaskKey(targetFullCodePath, task.Code))
	}
	return created, nil
}

func buildCapabilityBundleAgentTaskRequest(ctx context.Context, targetFullCodePath string, task *dto.CapabilityBundleAgentTask) (scheduledsdk.CreateTaskRequest, error) {
	if task == nil {
		return scheduledsdk.CreateTaskRequest{}, fmt.Errorf("Agent 任务不能为空")
	}
	targetFullCodePath = strings.TrimSpace(targetFullCodePath)
	if targetFullCodePath == "" {
		return scheduledsdk.CreateTaskRequest{}, fmt.Errorf("Agent 任务 %s 目标目录为空", task.Code)
	}
	code := normalizeCapabilityAgentTaskCode(task.Code)
	if code == "" {
		return scheduledsdk.CreateTaskRequest{}, fmt.Errorf("Agent 任务 code 不能为空")
	}
	message := strings.TrimSpace(task.Message)
	if message == "" {
		return scheduledsdk.CreateTaskRequest{}, fmt.Errorf("Agent 任务 %s message 不能为空", code)
	}
	schedule := task.Schedule
	if err := schedule.Validate(); err != nil {
		return scheduledsdk.CreateTaskRequest{}, fmt.Errorf("Agent 任务 %s 计划错误: %w", code, err)
	}
	modeCode := strings.TrimSpace(task.ModeCode)
	if modeCode == "" {
		modeCode = "dev"
	}
	executorPayload := map[string]interface{}{
		"full_code_path":  targetFullCodePath,
		"message":         message,
		"display_content": message,
	}
	if modeCode != "" && modeCode != "dev" {
		executorPayload["mode_code"] = modeCode
	}
	if task.MaxDurationSeconds > 0 {
		executorPayload["max_duration_seconds"] = task.MaxDurationSeconds
	}
	title := strings.TrimSpace(task.Title)
	if title == "" {
		title = code
	}
	status := scheduledsdk.TaskStatusPaused
	if task.Enabled {
		status = scheduledsdk.TaskStatusPending
	}
	requestUser := strings.TrimSpace(contextx.GetRequestUser(ctx))
	if requestUser == "" {
		requestUser = "system"
	}
	return scheduledsdk.CreateTaskRequest{
		Title:           title,
		Description:     strings.TrimSpace(task.Description),
		Category:        "scheduled_agent_session",
		Tags:            []string{"agent", "session", "capability_bundle"},
		IdempotencyKey:  capabilityBundleAgentTaskIdempotencyKey(targetFullCodePath, code),
		ExecutorKey:     ScheduledAgentSessionExecutorKey,
		ExecutorPayload: mustRawJSON(executorPayload),
		Metadata: map[string]string{
			"kind":             "scheduled_agent_session",
			"managed_by":       "capability_bundle",
			"bundle_task_code": code,
			"schedule_code":    code,
			"mode_code":        modeCode,
		},
		Status:          status,
		Schedule:        schedule,
		SourceType:      "agent_session",
		SourceRef:       targetFullCodePath,
		ResourceScope:   "workspace_directory",
		ResourceKey:     targetFullCodePath,
		RequestUser:     requestUser,
		RequestUserDept: contextx.GetRequestDepartmentFullPath(ctx),
		CreatedBy:       requestUser,
	}, nil
}

func capabilityBundleAgentTaskIdempotencyKey(fullCodePath string, code string) string {
	parts := strings.Join([]string{strings.TrimSpace(fullCodePath), strings.TrimSpace(code)}, "\x00")
	sum := sha1.Sum([]byte(parts))
	return "bundle-agent-task-" + hex.EncodeToString(sum[:])
}
