package apicall

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
)

// PublishDirectoryToHub 发布目录到 Hub
func PublishDirectoryToHub(ctx context.Context, req *dto.PublishHubDirectoryReq) (*dto.PublishHubDirectoryResp, error) {
	return PostAPI[*dto.PublishHubDirectoryReq, *dto.PublishHubDirectoryResp](ctx, "/hub/api/v1/directories/publish", req)
}

// UpdateDirectoryToHub 更新目录到 Hub（用于 push）
func UpdateDirectoryToHub(ctx context.Context, req *dto.UpdateHubDirectoryReq) (*dto.UpdateHubDirectoryResp, error) {
	return PutAPI[*dto.UpdateHubDirectoryReq, *dto.UpdateHubDirectoryResp](ctx, "/hub/api/v1/directories/update", req)
}

// GetHubDirectoryList 获取 Hub 目录列表
func GetHubDirectoryList(ctx context.Context, req *dto.GetHubDirectoryListReq) (*dto.HubDirectoryListResp, error) {
	// 构建查询参数
	path := "/hub/api/v1/directories"
	params := url.Values{}
	params.Set("page", strconv.Itoa(req.Page))
	params.Set("page_size", strconv.Itoa(req.PageSize))
	if req.Search != "" {
		params.Set("search", req.Search)
	}
	if req.Category != "" {
		params.Set("category", req.Category)
	}
	if req.PublisherUsername != "" {
		params.Set("publisher_username", req.PublisherUsername)
	}

	return GetAPI[*dto.HubDirectoryListResp](ctx, path, params)
}

// GetHubDirectoryDetail 获取 Hub 目录详情（通过网关，使用 full_code_path）
func GetHubDirectoryDetail(ctx context.Context, req *dto.GetHubDirectoryDetailReq) (*dto.HubDirectoryDetailDetailResp, error) {
	// 构建查询参数
	path := "/hub/api/v1/directories/detail"
	params := url.Values{}
	params.Set("full_code_path", req.FullCodePath)
	if req.Version != "" {
		params.Set("version", req.Version)
	}
	if req.IncludeTree {
		params.Set("include_tree", "true")
	}

	return GetAPI[*dto.HubDirectoryDetailDetailResp](ctx, path, params)
}

// GetHubDirectoryDetailFromHost 从指定的 Hub 主机获取目录详情（通过 full-code-path，支持版本号）
// 用于跨 Hub 主机调用，不通过网关
func GetHubDirectoryDetailFromHost(ctx context.Context, req *dto.GetHubDirectoryDetailFromHostReq) (*dto.HubDirectoryDetailDetailResp, error) {
	// 如果 ctx 为 nil，使用 Background（用于公开接口）
	if ctx == nil {
		ctx = context.Background()
	}

	// 构建 Hub API URL
	baseURL := req.Host
	if !strings.HasPrefix(req.Host, "http://") && !strings.HasPrefix(req.Host, "https://") {
		baseURL = "http://" + req.Host
	}

	// 构建查询参数
	path := "/hub/api/v1/directories/detail"
	params := url.Values{}
	params.Set("full_code_path", req.FullCodePath)
	if req.Version != "" {
		params.Set("version", req.Version)
	}
	if req.IncludeTree {
		params.Set("include_tree", "true")
	}

	fullURL := fmt.Sprintf("%s%s?%s", baseURL, path, params.Encode())

	// 使用 callAPIWithURL 调用（不需要 header，因为是公开接口，但保留 ctx 用于超时控制）
	result, err := callAPIWithURL[*dto.HubDirectoryDetailDetailResp](ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, err
	}
	return result.Data, nil
}

// CallAPIWithURL 使用完整 URL 调用 API（支持查询参数，公开方法）
// 注意：这里直接使用完整 URL，不通过 serviceconfig.GetGatewayURL()
func CallAPIWithURL[T any](ctx context.Context, method, fullURL string, reqBody interface{}) (*ApiResult[T], error) {
	return callAPIWithURL[T](ctx, method, fullURL, reqBody)
}

// callAPIWithURL 使用完整 URL 调用 API（支持查询参数，内部方法）
// 注意：这里直接使用完整 URL，不通过 serviceconfig.GetGatewayURL()
func callAPIWithURL[T any](ctx context.Context, method, fullURL string, reqBody interface{}) (*ApiResult[T], error) {
	// 如果 ctx 为 nil，使用 Background
	if ctx == nil {
		ctx = context.Background()
	}

	var bodyReader io.Reader
	if reqBody != nil {
		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("序列化请求体失败: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	
	// ✨ 从 ctx 中提取 Token
	if token := contextx.GetToken(ctx); token != "" {
		req.Header.Set("X-Token", token)
	}
	
	// ✨ 从 ctx 中提取追踪ID
	if traceID := contextx.GetTraceId(ctx); traceID != "" {
		req.Header.Set("X-Trace-Id", traceID)
	}

	// 注意：不需要设置 X-Request-User，因为：
	// 1. 如果请求经过网关，网关会从 X-Token 解析并覆盖 X-Request-User
	// 2. 如果请求不经过网关（如跨 Hub 调用），通常是公开接口，不需要用户信息

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP错误: %d %s, 响应: %s", resp.StatusCode, resp.Status, string(bodyBytes))
	}

	var result ApiResult[T]
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w, 响应内容: %s", err, string(bodyBytes))
	}

	if result.Code != 0 {
		return &result, fmt.Errorf("业务错误 [%d]: %s", result.Code, result.Msg)
	}

	return &result, nil
}
