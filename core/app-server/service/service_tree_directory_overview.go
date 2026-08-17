package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/access"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/pkg/scheduledsdk"
	"github.com/kageos/kageos/pkg/serviceconfig"
)

const (
	directoryOverviewTaskPageSize    = 100
	directoryOverviewMaxTasksPerKind = 240
)

type serviceTreeScheduleClient interface {
	ListTasks(context.Context, scheduledsdk.ListTasksRequest) (*scheduledsdk.ListTasksResponse, error)
}

var newServiceTreeScheduleClient = func() serviceTreeScheduleClient {
	return scheduledsdk.NewClient(scheduledsdk.Options{
		BaseURL: serviceconfig.BuildGatewayURL("/timer/api/v1"),
	})
}

func (s *ServiceTreeService) GetDirectoryOverview(ctx context.Context, req *dto.GetDirectoryOverviewReq) (*dto.GetDirectoryOverviewResp, error) {
	return s.queryView.GetDirectoryOverview(ctx, req)
}

func (q *serviceTreeQueryView) GetDirectoryOverview(ctx context.Context, req *dto.GetDirectoryOverviewReq) (*dto.GetDirectoryOverviewResp, error) {
	if req == nil {
		return nil, fmt.Errorf("目录概览请求不能为空")
	}

	rootPath := access.NormalizeResourcePath(req.FullCodePath)
	if rootPath == "" {
		return nil, fmt.Errorf("full_code_path 不能为空")
	}

	root, err := q.serviceTreeRepo.GetServiceTreeByFullPath(rootPath)
	if err != nil {
		return nil, fmt.Errorf("获取目录失败: %w", err)
	}
	if root.Type != model.ServiceTreeTypePackage {
		return nil, fmt.Errorf("目录概览仅支持 package 类型节点")
	}
	if _, err := q.appRepo.GetAppByID(root.AppID); err != nil {
		return nil, fmt.Errorf("获取应用信息失败: %w", err)
	}

	descendants, err := q.serviceTreeRepo.GetDescendantNodes(root.AppID, root.FullCodePath)
	if err != nil {
		return nil, fmt.Errorf("获取子资源失败: %w", err)
	}

	nodes := append([]*model.ServiceTree{root}, descendants...)
	nodes, err = q.filterReadableNodes(ctx, nodes)
	if err != nil {
		return nil, err
	}
	resourceByPath := make(map[string]*dto.DirectoryOverviewResource, len(nodes))
	var directories []*dto.DirectoryOverviewResource
	var functions []*dto.DirectoryOverviewResource
	stats := dto.DirectoryOverviewStats{}

	for _, node := range nodes {
		resource := directoryOverviewResourceFromNode(node)
		if resource == nil {
			continue
		}
		resourceByPath[resource.FullCodePath] = resource
		switch node.Type {
		case model.ServiceTreeTypePackage:
			directories = append(directories, resource)
			if node.FullCodePath != root.FullCodePath {
				stats.Directories++
			}
		case model.ServiceTreeTypeFunction:
			functions = append(functions, resource)
			stats.Functions++
			stats.TotalRunCount += node.RunCount
		case model.ServiceTreeTypeDocs:
			stats.Docs++
		}
	}

	resp := &dto.GetDirectoryOverviewResp{
		Directory: directoryOverviewResourceFromNode(root),
		Stats:     stats,
	}

	client := newServiceTreeScheduleClient()
	functionTasks, functionTotal, functionWarnings := q.loadDirectoryOverviewTasks(ctx, client, root.FullCodePath, functions, "function", ScheduledFunctionExecutorKey, "function", resourceByPath)
	agentTasks, agentTotal, agentWarnings := q.loadDirectoryOverviewTasks(ctx, client, root.FullCodePath, directories, "agent", "agent.session", "workspace_directory", resourceByPath)

	resp.ScheduledFunctionTasks = directoryOverviewLimitTasks(functionTasks, directoryOverviewMaxTasksPerKind)
	resp.ScheduledAgentTasks = directoryOverviewLimitTasks(agentTasks, directoryOverviewMaxTasksPerKind)
	resp.Stats.ScheduledFunctionTasks = functionTotal
	resp.Stats.ScheduledAgentTasks = agentTotal
	q.fillDirectoryOverviewRuntimeStats(&resp.Stats, append(functionTasks, agentTasks...))

	resp.Warnings = append(resp.Warnings, functionWarnings...)
	resp.Warnings = append(resp.Warnings, agentWarnings...)
	if len(functionTasks) > len(resp.ScheduledFunctionTasks) {
		resp.Warnings = append(resp.Warnings, fmt.Sprintf("函数任务较多，清单仅返回前 %d 个", directoryOverviewMaxTasksPerKind))
	}
	if len(agentTasks) > len(resp.ScheduledAgentTasks) {
		resp.Warnings = append(resp.Warnings, fmt.Sprintf("Agent 任务较多，清单仅返回前 %d 个", directoryOverviewMaxTasksPerKind))
	}
	resp.Partial = len(resp.Warnings) > 0

	return resp, nil
}

func (q *serviceTreeQueryView) filterReadableNodes(ctx context.Context, nodes []*model.ServiceTree) ([]*model.ServiceTree, error) {
	if q.permission == nil {
		return nodes, nil
	}
	permissionsByPath, err := q.permissionsByPath(ctx, nodes)
	if err != nil {
		return nil, fmt.Errorf("计算目录权限失败: %w", err)
	}
	readable := make([]*model.ServiceTree, 0, len(nodes))
	for _, node := range nodes {
		if node == nil {
			continue
		}
		result := permissionsByPath[access.NormalizeResourcePath(node.FullCodePath)]
		if result != nil && result.Permissions[access.ActionRead] {
			readable = append(readable, node)
		}
	}
	return readable, nil
}

func (q *serviceTreeQueryView) loadDirectoryOverviewTasks(
	ctx context.Context,
	client serviceTreeScheduleClient,
	rootPath string,
	resources []*dto.DirectoryOverviewResource,
	kind string,
	executorKey string,
	resourceScope string,
	resourceByPath map[string]*dto.DirectoryOverviewResource,
) ([]*dto.DirectoryOverviewScheduledTask, int, []string) {
	if len(resources) == 0 {
		return nil, 0, nil
	}

	items := make([]*dto.DirectoryOverviewScheduledTask, 0)
	total := 0
	warnings := make([]string, 0)
	rootPath = access.NormalizeResourcePath(rootPath)
	for page := 1; len(items) < directoryOverviewMaxTasksPerKind; page++ {
		resp, err := client.ListTasks(ctx, scheduledsdk.ListTasksRequest{
			ExecutorKey:       executorKey,
			ResourceScope:     resourceScope,
			ResourceKeyPrefix: rootPath,
			Page:              page,
			PageSize:          directoryOverviewTaskPageSize,
		})
		if err != nil {
			logger.Errorf(ctx, "[DirectoryOverview] list scheduled tasks failed: endpoint=%s root_path=%s kind=%s executor_key=%s resource_scope=%s page=%d page_size=%d error=%v",
				serviceconfig.BuildGatewayURL("/timer/api/v1/tasks"), rootPath, kind, executorKey, resourceScope, page, directoryOverviewTaskPageSize, err)
			return items, total, []string{fmt.Sprintf("%s 定时配置加载失败: %v", rootPath, err)}
		}
		if resp == nil {
			break
		}
		total = int(resp.Total)
		for _, task := range resp.List {
			if task == nil || len(items) >= directoryOverviewMaxTasksPerKind {
				continue
			}
			resourcePath := access.NormalizeResourcePath(task.ResourceKey)
			if resourcePath == "" {
				resourcePath = rootPath
			}
			taskResource := resourceByPath[resourcePath]
			if taskResource == nil {
				taskResource = &dto.DirectoryOverviewResource{
					Name:         directoryOverviewResourceDisplayName("", resourcePath),
					FullCodePath: resourcePath,
				}
			}
			items = append(items, &dto.DirectoryOverviewScheduledTask{
				Kind:         kind,
				Origin:       scheduledTaskOrigin(task),
				Builtin:      isBuiltinScheduledTask(task),
				Resource:     taskResource,
				ResourcePath: taskResource.FullCodePath,
				ResourceName: taskResource.Name,
				Task:         directoryOverviewCompactTask(task),
			})
		}
		if len(resp.List) == 0 || len(items) >= total {
			break
		}
	}
	if total > len(items) {
		warnings = append(warnings, fmt.Sprintf("%s 下还有 %d 个定时配置未展开", rootPath, total-len(items)))
	}
	sort.SliceStable(items, func(i, j int) bool {
		return directoryOverviewTaskLess(items[i], items[j])
	})
	return items, total, warnings
}

func directoryOverviewResourceFromNode(node *model.ServiceTree) *dto.DirectoryOverviewResource {
	if node == nil {
		return nil
	}
	return &dto.DirectoryOverviewResource{
		ID:           node.ID,
		Name:         node.Name,
		Code:         node.Code,
		Type:         node.Type,
		FullCodePath: node.FullCodePath,
		TemplateType: node.TemplateType,
		RunCount:     node.RunCount,
	}
}

func directoryOverviewCompactTask(task *scheduledsdk.Task) *scheduledsdk.Task {
	if task == nil {
		return nil
	}
	compact := *task
	if strings.TrimSpace(compact.Description) == "" {
		compact.Description = directoryOverviewTaskPayloadMessage(compact.ExecutorPayload)
	}
	compact.ExecutorPayload = nil
	compact.Metadata = nil
	compact.Tags = nil
	compact.IdempotencyKey = ""
	compact.Category = ""
	compact.SourceType = ""
	compact.SourceRef = ""
	compact.RequestUser = ""
	compact.RequestUserDept = ""
	compact.CreatedBy = ""
	return &compact
}

func directoryOverviewTaskPayloadMessage(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	payload := struct {
		Message string `json:"message"`
	}{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ""
	}
	return strings.TrimSpace(payload.Message)
}

func directoryOverviewLimitTasks(tasks []*dto.DirectoryOverviewScheduledTask, limit int) []*dto.DirectoryOverviewScheduledTask {
	if limit <= 0 || len(tasks) <= limit {
		return tasks
	}
	return tasks[:limit]
}

func (q *serviceTreeQueryView) fillDirectoryOverviewRuntimeStats(stats *dto.DirectoryOverviewStats, tasks []*dto.DirectoryOverviewScheduledTask) {
	for _, item := range tasks {
		if item == nil || item.Task == nil {
			continue
		}
		task := item.Task
		if task.InflightExecutionID > 0 {
			stats.RunningTasks++
		}
		if task.Status == scheduledsdk.TaskStatusFailed {
			stats.FailedTasks++
		}
		if task.Status == scheduledsdk.TaskStatusPaused {
			stats.PausedTasks++
		}
		if task.NextRunAt != nil && task.Status == scheduledsdk.TaskStatusPending {
			if stats.NextRunAt == nil || task.NextRunAt.Before(*stats.NextRunAt) {
				next := *task.NextRunAt
				stats.NextRunAt = &next
			}
		}
	}
}

func directoryOverviewTaskLess(left, right *dto.DirectoryOverviewScheduledTask) bool {
	if left == nil || left.Task == nil {
		return false
	}
	if right == nil || right.Task == nil {
		return true
	}
	leftRunning := left.Task.InflightExecutionID > 0
	rightRunning := right.Task.InflightExecutionID > 0
	if leftRunning != rightRunning {
		return leftRunning
	}

	leftTime := directoryOverviewTaskTime(left.Task.NextRunAt)
	rightTime := directoryOverviewTaskTime(right.Task.NextRunAt)
	if !leftTime.Equal(rightTime) {
		return leftTime.Before(rightTime)
	}
	return left.Task.ID > right.Task.ID
}

func directoryOverviewTaskTime(value *time.Time) time.Time {
	if value == nil || value.IsZero() {
		return time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
	}
	return *value
}

func directoryOverviewResourceDisplayName(name, path string) string {
	name = strings.TrimSpace(name)
	if name != "" {
		return name
	}
	parts := strings.Split(strings.Trim(access.NormalizeResourcePath(path), "/"), "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return strings.TrimSpace(path)
}
