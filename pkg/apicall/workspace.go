package apicall

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/ai-agent-os/ai-agent-os/dto"
)

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
