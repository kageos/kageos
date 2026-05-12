package apicall

import (
	"context"
	"fmt"

	workflowdto "github.com/ai-agent-os/ai-agent-os/core/workflow-server/dto"
	"github.com/ai-agent-os/ai-agent-os/dto"
)

// CreateWorkflowNode 创建 workflow 类型服务树节点（agent-server -> app-server）。
func CreateWorkflowNode(ctx context.Context, req *dto.CreateWorkflowNodeReq) (*dto.CreateWorkflowNodeResp, error) {
	return PostAPI[*dto.CreateWorkflowNodeReq, *dto.CreateWorkflowNodeResp](ctx, "/workspace/api/v1/workflows/crud", req)
}

// CreateWorkflowDefinition 创建 workflow-server 草稿定义（agent-server -> workflow-server）。
func CreateWorkflowDefinition(ctx context.Context, req workflowdto.CreateWorkflowRequest) (*workflowdto.WorkflowItem, error) {
	return PostAPI[workflowdto.CreateWorkflowRequest, *workflowdto.WorkflowItem](ctx, "/workflow/api/v1/workflows", req)
}

// GetWorkflowDefinitionByFullCodePath 按 full_code_path 查询 workflow-server 定义。
func GetWorkflowDefinitionByFullCodePath(ctx context.Context, fullCodePath string) (*workflowdto.WorkflowItem, error) {
	return GetAPI[*workflowdto.WorkflowItem](ctx, "/workflow/api/v1/workflows/by_path", buildQueryParams(
		withFullCodePathQuery(fullCodePath),
	))
}

// UpdateWorkflowDefinition 更新 workflow-server 草稿定义。
func UpdateWorkflowDefinition(ctx context.Context, id int64, req workflowdto.UpdateWorkflowRequest) (*workflowdto.WorkflowItem, error) {
	path := fmt.Sprintf("/workflow/api/v1/workflows/%d", id)
	return PutAPI[workflowdto.UpdateWorkflowRequest, *workflowdto.WorkflowItem](ctx, path, req)
}

// PublishWorkflowDefinition 发布 workflow-server 定义版本。
func PublishWorkflowDefinition(ctx context.Context, id int64, req workflowdto.PublishWorkflowRequest) (*workflowdto.WorkflowVersionItem, error) {
	path := fmt.Sprintf("/workflow/api/v1/workflows/%d/publish", id)
	return PostAPI[workflowdto.PublishWorkflowRequest, *workflowdto.WorkflowVersionItem](ctx, path, req)
}
