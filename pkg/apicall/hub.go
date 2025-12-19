package apicall

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/serviceconfig"
)

// PublishDirectoryToHub 发布目录到 Hub
func PublishDirectoryToHub(header *Header, req *dto.PublishHubDirectoryReq) (*dto.PublishHubDirectoryResp, error) {
	result, err := callAPI[dto.PublishHubDirectoryResp](
		http.MethodPost,
		"/hub/api/v1/directories/publish",
		header,
		req,
	)
	if err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// UpdateDirectoryToHub 更新目录到 Hub（用于 push）
func UpdateDirectoryToHub(header *Header, req *dto.UpdateHubDirectoryReq) (*dto.UpdateHubDirectoryResp, error) {
	result, err := callAPI[dto.UpdateHubDirectoryResp](
		http.MethodPut,
		"/hub/api/v1/directories/update",
		header,
		req,
	)
	if err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// GetHubDirectoryList 获取 Hub 目录列表
func GetHubDirectoryList(header *Header, page, pageSize int, search, category, publisherUsername string) (*dto.HubDirectoryListResp, error) {
	// 构建查询参数
	path := "/hub/api/v1/directories"
	params := url.Values{}
	params.Set("page", strconv.Itoa(page))
	params.Set("page_size", strconv.Itoa(pageSize))
	if search != "" {
		params.Set("search", search)
	}
	if category != "" {
		params.Set("category", category)
	}
	if publisherUsername != "" {
		params.Set("publisher_username", publisherUsername)
	}

	// 构建完整 URL
	gatewayURL := serviceconfig.GetGatewayURL()
	fullURL := fmt.Sprintf("%s%s?%s", gatewayURL, path, params.Encode())

	// 使用 callAPIWithURL 调用（需要新增这个函数，或者直接构建 URL 调用）
	result, err := callAPIWithURL[dto.HubDirectoryListResp](
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

// GetHubDirectoryDetail 获取 Hub 目录详情（通过网关，使用 full_code_path）
func GetHubDirectoryDetail(header *Header, fullCodePath string, includeTree, includeFiles bool) (*dto.HubDirectoryDetailDetailResp, error) {
	// 构建查询参数
	path := "/hub/api/v1/directories/detail"
	params := url.Values{}
	params.Set("full_code_path", fullCodePath)
	if includeTree {
		params.Set("include_tree", "true")
	}
	if includeFiles {
		params.Set("include_files", "true")
	}

	// 构建完整 URL
	gatewayURL := serviceconfig.GetGatewayURL()
	fullURL := fmt.Sprintf("%s%s?%s", gatewayURL, path, params.Encode())

	result, err := callAPIWithURL[dto.HubDirectoryDetailDetailResp](
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

// GetHubDirectoryDetailFromHost 从指定的 Hub 主机获取目录详情（通过 full-code-path）
// 用于跨 Hub 主机调用，不通过网关
func GetHubDirectoryDetailFromHost(host string, fullCodePath string, includeTree, includeFiles bool) (*dto.HubDirectoryDetailDetailResp, error) {
	// 构建 Hub API URL
	baseURL := host
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		baseURL = "http://" + host
	}

	// 构建查询参数
	path := "/hub/api/v1/directories/detail"
	params := url.Values{}
	params.Set("full_code_path", fullCodePath)
	if includeTree {
		params.Set("include_tree", "true")
	}
	if includeFiles {
		params.Set("include_files", "true")
	}

	fullURL := fmt.Sprintf("%s%s?%s", baseURL, path, params.Encode())

	// 使用 callAPIWithURL 调用（不需要 header，因为是公开接口）
	result, err := callAPIWithURL[dto.HubDirectoryDetailDetailResp](
		http.MethodGet,
		fullURL,
		nil, // 不需要 header
		nil,
	)
	if err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// CallAPIWithURL 使用完整 URL 调用 API（支持查询参数，公开方法）
// 注意：这里直接使用完整 URL，不通过 serviceconfig.GetGatewayURL()
func CallAPIWithURL[T any](method, fullURL string, header *Header, reqBody interface{}) (*ApiResult[T], error) {
	return callAPIWithURL[T](method, fullURL, header, reqBody)
}

// callAPIWithURL 使用完整 URL 调用 API（支持查询参数，内部方法）
// 注意：这里直接使用完整 URL，不通过 serviceconfig.GetGatewayURL()
func callAPIWithURL[T any](method, fullURL string, header *Header, reqBody interface{}) (*ApiResult[T], error) {
	var bodyReader io.Reader
	if reqBody != nil {
		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("序列化请求体失败: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequest(method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if header != nil && header.Token != "" {
		req.Header.Set("X-Token", header.Token)
	}
	if header != nil && header.TraceID != "" {
		req.Header.Set("X-Trace-Id", header.TraceID)
	}
	if header != nil && header.RequestUser != "" {
		req.Header.Set("X-Request-User", header.RequestUser)
	}

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
