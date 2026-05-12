package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/core/workflow-server/definition"
	workflowdto "github.com/ai-agent-os/ai-agent-os/core/workflow-server/dto"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/pkg/servicetree"
)

const workflowCodeSuffix = ".workflow"

type CreateWorkflowTool struct{}

type createWorkflowArgs struct {
	FullCodePath string `json:"full_code_path" schema_desc:"工作流节点完整路径，例如 /user/app/workflows/customer_onboarding.workflow" schema_required:"true"`
	Name         string `json:"name" schema_desc:"工作流显示名称" schema_required:"true"`
	Description  string `json:"description" schema_desc:"工作流描述"`
	Definition   string `json:"definition" schema_desc:"workflow.v1 definition JSON 字符串；必须把 schema_version 放在顶层，不要外包 workflow_name/definition；start/output 必须作为节点声明 schema" schema_required:"true"`
	Publish      bool   `json:"publish" schema_desc:"是否创建后立即发布；默认 false，发布要求 nodes 非空且图定义完整"`
}

type createWorkflowResultData struct {
	WorkflowID      int64  `json:"workflow_id" schema_desc:"workflow-server 定义 ID" schema_required:"true"`
	TreeNodeID      int64  `json:"tree_node_id" schema_desc:"服务树节点 ID" schema_required:"true"`
	Name            string `json:"name" schema_desc:"工作流名称" schema_required:"true"`
	FullCodePath    string `json:"full_code_path" schema_desc:"工作流完整路径" schema_required:"true"`
	AppID           int64  `json:"app_id" schema_desc:"应用 ID" schema_required:"true"`
	Status          string `json:"status" schema_desc:"工作流状态" schema_required:"true"`
	LatestVersionID int64  `json:"latest_version_id,omitempty" schema_desc:"最新发布版本 ID"`
	Published       bool   `json:"published" schema_desc:"本次是否发布" schema_required:"true"`
	VersionID       int64  `json:"version_id,omitempty" schema_desc:"本次发布版本 ID"`
	Version         int    `json:"version,omitempty" schema_desc:"本次发布版本号"`
}

type workflowPathParts struct {
	FullCodePath string
	ParentPath   string
	User         string
	App          string
	Code         string
}

var createWorkflowToolDef = toolDefinitionWithOutput[createWorkflowArgs, structuredToolResultSchema[createWorkflowResultData]](
	"create_workflow",
	"创建或更新一个工作流：先确保服务树上有 workflow 节点，再把 workflow.v1 definition 写入 workflow-server。definition 必须是工作流定义本体，不能包 workflow_name/definition。",
)

func (t *CreateWorkflowTool) Definition() dto.ToolDef {
	return createWorkflowToolDef
}

func (t *CreateWorkflowTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[createWorkflowArgs](call.Args)
	if err != nil {
		return toolResult("create_workflow 参数解析失败: "+err.Error(), true)
	}
	data, notice, err := runCreateWorkflowTool(ctx, args, call.FullCodePath)
	if err != nil {
		return toolResult("create_workflow 失败: "+err.Error(), true)
	}
	return toolResultWithStructuredData(data, false, notice)
}

func runCreateWorkflowTool(ctx context.Context, args createWorkflowArgs, currentFullCodePath string) (createWorkflowResultData, string, error) {
	var zero createWorkflowResultData

	name := strings.TrimSpace(args.Name)
	if name == "" {
		return zero, "", fmt.Errorf("name is required")
	}
	rawDefinition, err := normalizeWorkflowDefinitionRaw(args.Definition)
	if err != nil {
		return zero, "", err
	}
	if err := validateWorkflowDefinitionForTool(rawDefinition, args.Publish); err != nil {
		return zero, "", err
	}

	parts, err := parseWorkflowFullCodePath(resolveFullCodePathArg(args.FullCodePath, currentFullCodePath))
	if err != nil {
		return zero, "", err
	}

	ctx = withAgentToolClientSource(ctx)
	treeNode, err := ensureWorkflowTreeNode(ctx, parts, name, strings.TrimSpace(args.Description))
	if err != nil {
		return zero, "", err
	}

	workflow, err := upsertWorkflowDefinition(ctx, parts.FullCodePath, name, strings.TrimSpace(args.Description), treeNode.AppID, rawDefinition)
	if err != nil {
		return zero, "", err
	}

	result := createWorkflowResultData{
		WorkflowID:      workflow.ID,
		TreeNodeID:      treeNode.ID,
		Name:            workflow.Name,
		FullCodePath:    workflow.FullCodePath,
		AppID:           workflow.AppID,
		Status:          workflow.Status,
		LatestVersionID: workflow.LatestVersionID,
	}

	if args.Publish {
		version, err := apicall.PublishWorkflowDefinition(ctx, workflow.ID, workflowdto.PublishWorkflowRequest{
			Definition: rawDefinition,
		})
		if err != nil {
			return zero, "", err
		}
		result.Published = true
		result.VersionID = version.ID
		result.Version = version.Version
		result.Status = "enabled"
		result.LatestVersionID = version.ID
	}

	notice := fmt.Sprintf("工作流已创建/更新：%s。可在左侧服务树点击该 workflow 节点查看只读流程图。", result.FullCodePath)
	if result.Published {
		notice = fmt.Sprintf("工作流已创建并发布：%s（v%d）。可在左侧服务树点击该 workflow 节点查看只读流程图。", result.FullCodePath, result.Version)
	}
	return result, notice, nil
}

func ensureWorkflowTreeNode(ctx context.Context, parts workflowPathParts, name string, description string) (*dto.CreateWorkflowNodeResp, error) {
	detail, err := apicall.GetServiceTreeDetailByFullCodePath(ctx, parts.FullCodePath)
	if err == nil && detail != nil {
		if detail.Type != servicetree.TypeWorkflow {
			return nil, fmt.Errorf("路径 %s 已存在但类型是 %s，不是 workflow", parts.FullCodePath, detail.Type)
		}
		return &dto.CreateWorkflowNodeResp{
			ID:           detail.ID,
			Name:         detail.Name,
			Code:         detail.Code,
			Type:         detail.Type,
			Description:  detail.Description,
			AppID:        detail.AppID,
			FullCodePath: detail.FullCodePath,
		}, nil
	}

	parent, parentErr := apicall.GetServiceTreeDetailByFullCodePath(ctx, parts.ParentPath)
	if parentErr != nil || parent == nil {
		logger.Warnf(ctx, "[CreateWorkflowTool] workflow 父目录不存在 - ParentPath: %s, error: %v", parts.ParentPath, parentErr)
		return nil, fmt.Errorf("父目录不存在: %s，请先创建目录后再创建 workflow", parts.ParentPath)
	}

	req := &dto.CreateWorkflowNodeReq{
		User:               parts.User,
		App:                parts.App,
		Name:               name,
		Code:               parts.Code,
		ParentFullCodePath: parts.ParentPath,
		Description:        description,
	}
	resp, err := apicall.CreateWorkflowNode(ctx, req)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			existing, getErr := apicall.GetServiceTreeDetailByFullCodePath(ctx, parts.FullCodePath)
			if getErr == nil && existing != nil && existing.Type == servicetree.TypeWorkflow {
				return &dto.CreateWorkflowNodeResp{
					ID:           existing.ID,
					Name:         existing.Name,
					Code:         existing.Code,
					Type:         existing.Type,
					Description:  existing.Description,
					AppID:        existing.AppID,
					FullCodePath: existing.FullCodePath,
				}, nil
			}
		}
		return nil, err
	}
	return resp, nil
}

func upsertWorkflowDefinition(ctx context.Context, fullCodePath, name, description string, appID int64, rawDefinition json.RawMessage) (*workflowdto.WorkflowItem, error) {
	existing, err := apicall.GetWorkflowDefinitionByFullCodePath(ctx, fullCodePath)
	if err != nil {
		return nil, err
	}

	if existing == nil || existing.ID == 0 {
		existing, err = apicall.CreateWorkflowDefinition(ctx, workflowdto.CreateWorkflowRequest{
			Name:         name,
			Description:  description,
			AppID:        appID,
			FullCodePath: fullCodePath,
			Definition:   rawDefinition,
		})
		if err != nil {
			return nil, err
		}
	}

	namePtr := name
	descriptionPtr := description
	appIDPtr := appID
	fullCodePathPtr := fullCodePath
	rawCopy := json.RawMessage(append([]byte(nil), rawDefinition...))
	return apicall.UpdateWorkflowDefinition(ctx, existing.ID, workflowdto.UpdateWorkflowRequest{
		Name:         &namePtr,
		Description:  &descriptionPtr,
		AppID:        &appIDPtr,
		FullCodePath: &fullCodePathPtr,
		Definition:   &rawCopy,
	})
}

func normalizeWorkflowDefinitionRaw(raw string) (json.RawMessage, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, fmt.Errorf("definition is required")
	}

	var obj map[string]json.RawMessage
	if err := json.Unmarshal([]byte(trimmed), &obj); err != nil {
		return nil, fmt.Errorf("definition 不是合法 JSON Object: %w", err)
	}
	if obj == nil {
		return nil, fmt.Errorf("definition 必须是 JSON Object")
	}
	if _, hasWrapper := obj["definition"]; hasWrapper {
		if _, hasSchemaVersion := obj["schema_version"]; !hasSchemaVersion {
			return nil, fmt.Errorf("definition 参数必须直接传 workflow.v1 定义本体，不要传 {workflow_name, definition} 这种外层包装")
		}
	}
	var schemaVersion string
	if rawVersion, ok := obj["schema_version"]; ok {
		_ = json.Unmarshal(rawVersion, &schemaVersion)
	}
	if strings.TrimSpace(schemaVersion) != definition.SchemaVersionV1 {
		return nil, fmt.Errorf("unsupported schema_version: %s", strings.TrimSpace(schemaVersion))
	}

	var compact bytes.Buffer
	if err := json.Compact(&compact, []byte(trimmed)); err != nil {
		return nil, fmt.Errorf("definition 压缩失败: %w", err)
	}
	return json.RawMessage(compact.Bytes()), nil
}

func validateWorkflowDefinitionForTool(raw json.RawMessage, publish bool) error {
	parsed, err := definition.Parse(raw)
	if err != nil {
		return err
	}
	if err := parsed.Validate(definition.ValidateOptions{SupportedNodeTypes: definition.SupportedMVPNodeTypes()}); err != nil {
		return err
	}
	return nil
}

func parseWorkflowFullCodePath(fullCodePath string) (workflowPathParts, error) {
	fullCodePath = normalizeAbsoluteToolPath(fullCodePath)
	fullCodePath = strings.TrimSpace(fullCodePath)
	if fullCodePath == "" {
		return workflowPathParts{}, fmt.Errorf("full_code_path is required")
	}

	trimmed := strings.Trim(fullCodePath, "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) < 3 {
		return workflowPathParts{}, fmt.Errorf("full_code_path 至少需要 /user/app/code.workflow 三段")
	}

	code := strings.TrimSpace(parts[len(parts)-1])
	if code == "" {
		return workflowPathParts{}, fmt.Errorf("full_code_path 末段 code 不能为空")
	}
	if !strings.HasSuffix(code, workflowCodeSuffix) {
		code += workflowCodeSuffix
		parts[len(parts)-1] = code
	}

	normalizedFullPath := "/" + strings.Join(parts, "/")
	parentPath := "/" + strings.Join(parts[:len(parts)-1], "/")
	return workflowPathParts{
		FullCodePath: normalizedFullPath,
		ParentPath:   parentPath,
		User:         parts[0],
		App:          parts[1],
		Code:         code,
	}, nil
}
