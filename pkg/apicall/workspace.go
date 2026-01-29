package apicall

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/ai-agent-os/ai-agent-os/dto"
)

// GetServiceTreeDetailByFullCodePath 根据 full_code_path 获取服务目录详情（agent-server -> app-server）
// 用于从 full_code_path 解析出节点信息（含 id/tree_id），供 add_functions、权限等使用
func GetServiceTreeDetailByFullCodePath(ctx context.Context, fullCodePath string) (*dto.GetServiceTreeDetailResp, error) {
	params := url.Values{}
	params.Set("full_code_path", fullCodePath)
	return GetAPI[*dto.GetServiceTreeDetailResp](ctx, "/workspace/api/v1/service_tree/detail", params)
}

// ServiceTreeAddFunctions 向服务目录添加函数（agent-server -> workspace）
// 将生成的代码写入到工作空间对应的目录下，并更新工作空间
// async: true 表示异步处理（通过回调通知），false 表示同步处理（直接返回结果）
func ServiceTreeAddFunctions(ctx context.Context, req *dto.AddFunctionsReq) (*dto.AddFunctionsResp, error) {
	return PostAPI[*dto.AddFunctionsReq, *dto.AddFunctionsResp](ctx, "/workspace/api/v1/service_tree/add_functions", req)
}

// SearchFunctions 搜索函数（agent-server -> app-server）
// 根据关键词、类型等条件搜索函数，支持分页
func SearchFunctions(ctx context.Context, req *dto.SearchFunctionsReq) (*dto.SearchFunctionsResp, error) {
	// 构建查询参数
	path := "/workspace/api/v1/service_tree/search_functions"
	params := url.Values{}
	params.Set("page", strconv.Itoa(req.Page))
	params.Set("page_size", strconv.Itoa(req.PageSize))
	if req.User != "" {
		params.Set("user", req.User)
	}
	if req.App != "" {
		params.Set("app", req.App)
	}
	if req.Keyword != "" {
		params.Set("keyword", req.Keyword)
	}
	if req.TemplateType != "" {
		params.Set("template_type", req.TemplateType)
	}

	return GetAPI[*dto.SearchFunctionsResp](ctx, path, params)
}

// CreateServiceTree 创建服务目录（agent-server -> app-server）
func CreateServiceTree(ctx context.Context, req *dto.CreateServiceTreeReq) (*dto.CreateServiceTreeResp, error) {
	return PostAPI[*dto.CreateServiceTreeReq, *dto.CreateServiceTreeResp](ctx, "/workspace/api/v1/service_tree", req)
}

// GetServiceTreeByID 根据ID获取服务目录（agent-server -> app-server）
func GetServiceTreeByID(ctx context.Context, req *dto.GetServiceTreeByIDReq) (*dto.GetServiceTreeResp, error) {
	path := fmt.Sprintf("/workspace/api/v1/service_tree/%d", req.ID)
	return GetAPI[*dto.GetServiceTreeResp](ctx, path, nil)
}

// GetWorkspaceContext 获取工作台环境信息（agent-server -> app-server）
func GetWorkspaceContext(ctx context.Context, fullCodePath string) (*dto.GetWorkspaceContextResp, error) {
	params := url.Values{}
	params.Set("full_code_path", fullCodePath)
	return GetAPI[*dto.GetWorkspaceContextResp](ctx, "/workspace/api/v1/workspace/context", params)
}

// CreateDocs 创建 docs 类型节点（agent-server -> app-server）
// 使用现有接口 POST /workspace/api/v1/docs/crud
func CreateDocs(ctx context.Context, req *dto.CreateDocsReq) (*dto.CreateDocsResp, error) {
	return PostAPI[*dto.CreateDocsReq, *dto.CreateDocsResp](ctx, "/workspace/api/v1/docs/crud", req)
}

// UpdateDocs 更新 docs 类型节点（含文档内容）（agent-server -> app-server）
// 使用现有接口 PUT /workspace/api/v1/docs/crud/:id
func UpdateDocs(ctx context.Context, id int64, req *dto.UpdateDocsReq) error {
	path := fmt.Sprintf("/workspace/api/v1/docs/crud/%d", id)
	req.ID = id
	_, err := PutAPI[*dto.UpdateDocsReq, map[string]interface{}](ctx, path, req)
	return err
}

// CreatePackage 创建 package 类型节点（目录）（agent-server -> app-server）
// 使用现有接口 POST /workspace/api/v1/packages
func CreatePackage(ctx context.Context, req *dto.CreatePackageReq) (*dto.CreatePackageResp, error) {
	return PostAPI[*dto.CreatePackageReq, *dto.CreatePackageResp](ctx, "/workspace/api/v1/packages", req)
}
