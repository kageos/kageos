package apicall

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/serviceconfig"
)

// ServiceTreeAddFunctions 向服务目录添加函数（agent-server -> workspace）
// 将生成的代码写入到工作空间对应的目录下，并更新工作空间
// async: true 表示异步处理（通过回调通知），false 表示同步处理（直接返回结果）
func ServiceTreeAddFunctions(header *Header, req *dto.AddFunctionsReq) (*dto.AddFunctionsResp, error) {
	result, err := callAPI[dto.AddFunctionsResp](
		http.MethodPost,
		"/workspace/api/v1/service_tree/add_functions",
		header,
		req,
	)
	if err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// SearchFunctions 搜索函数（agent-server -> app-server）
// 根据关键词、类型等条件搜索函数，支持分页
func SearchFunctions(header *Header, req *dto.SearchFunctionsReq) (*dto.SearchFunctionsResp, error) {
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

	// 构建完整 URL
	gatewayURL := serviceconfig.GetGatewayURL()
	fullURL := fmt.Sprintf("%s%s?%s", gatewayURL, path, params.Encode())

	// 使用 callAPIWithURL 调用
	result, err := callAPIWithURL[dto.SearchFunctionsResp](
		http.MethodGet,
		fullURL,
		header,
		nil,
	)
	if err != nil {
		return nil, err
	}
	return &result.Data, nil
}
