package apicall

import (
	"context"
	"net/http"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
)

// PublishDirectoryToHub 发布目录到 Hub（本地 Hub，走网关）
func PublishDirectoryToHub(ctx context.Context, req *dto.PublishHubDirectoryReq) (*dto.PublishHubDirectoryResp, error) {
	return PostAPI[*dto.PublishHubDirectoryReq, *dto.PublishHubDirectoryResp](ctx, "/hub/api/v1/directories/publish", req)
}

// UpdateDirectoryToHub 更新目录到 Hub（本地 Hub，走网关）
func UpdateDirectoryToHub(ctx context.Context, req *dto.UpdateHubDirectoryReq) (*dto.UpdateHubDirectoryResp, error) {
	return PutAPI[*dto.UpdateHubDirectoryReq, *dto.UpdateHubDirectoryResp](ctx, "/hub/api/v1/directories/update", req)
}

// PublishDirectoryToRemoteHub 发布目录到远程 Hub（跨站，用 Pub Key 认证）
func PublishDirectoryToRemoteHub(ctx context.Context, remoteURL, pubKey string, req *dto.PublishHubDirectoryReq) (*dto.PublishHubDirectoryResp, error) {
	fullURL := buildHubAPIURL(remoteURL, "/hub/api/v1/directories/publish", nil)
	return callAPIWithPubKey[*dto.PublishHubDirectoryResp](ctx, http.MethodPost, fullURL, pubKey, req)
}

// UpdateDirectoryToRemoteHub 更新目录到远程 Hub（跨站，用 Pub Key 认证）
func UpdateDirectoryToRemoteHub(ctx context.Context, remoteURL, pubKey string, req *dto.UpdateHubDirectoryReq) (*dto.UpdateHubDirectoryResp, error) {
	fullURL := buildHubAPIURL(remoteURL, "/hub/api/v1/directories/update", nil)
	return callAPIWithPubKey[*dto.UpdateHubDirectoryResp](ctx, http.MethodPut, fullURL, pubKey, req)
}

// callAPIWithPubKey 使用 Pub Key 认证调用远程 API
func callAPIWithPubKey[T any](ctx context.Context, method, fullURL, pubKey string, reqBody interface{}) (T, error) {
	var zero T
	result, err := callAPIWithOptions[T](ctx, method, fullURL, reqBody, withHeader(contextx.PubKeyHerder, pubKey))
	if err != nil {
		return zero, err
	}
	return result.Data, nil
}

// GetHubDirectoryList 获取 Hub 目录列表
func GetHubDirectoryList(ctx context.Context, req *dto.GetHubDirectoryListReq) (*dto.HubDirectoryListResp, error) {
	return GetAPI[*dto.HubDirectoryListResp](ctx, "/hub/api/v1/directories", buildQueryParams(
		withPaginationQuery(req.Page, req.PageSize),
		withTrimmedQueryValue("search", req.Search),
		withTrimmedQueryValue("category", req.Category),
		withTrimmedQueryValue("publisher_username", req.PublisherUsername),
	))
}

// GetHubDirectoryDetail 获取 Hub 目录详情（通过网关，支持 hub_directory_id 或 full_code_path）
// 有 HubDirectoryID 时优先用 ID 查（复制目录后从 b 推送时用 ID 才能命中原来从 a 发布的记录）
func GetHubDirectoryDetail(ctx context.Context, req *dto.GetHubDirectoryDetailReq) (*dto.HubDirectoryDetailDetailResp, error) {
	options := []queryOption{
		withVersionQuery(req.Version),
		withIncludeTreeQuery(req.IncludeTree),
	}
	if req.HubDirectoryID > 0 {
		options = append(options, withPositiveInt64QueryValue("hub_directory_id", req.HubDirectoryID))
	} else {
		options = append(options, withFullCodePathQuery(req.FullCodePath))
	}
	return GetAPI[*dto.HubDirectoryDetailDetailResp](ctx, "/hub/api/v1/directories/detail", buildQueryParams(options...))
}

// GetHubDirectoryDetailFromHost 从指定的 Hub 主机获取目录详情（通过 full-code-path，支持版本号）
// 用于跨 Hub 主机调用，不通过网关
func GetHubDirectoryDetailFromHost(ctx context.Context, req *dto.GetHubDirectoryDetailFromHostReq) (*dto.HubDirectoryDetailDetailResp, error) {
	fullURL := buildHubAPIURL(req.Host, "/hub/api/v1/directories/detail", buildQueryParams(
		withFullCodePathQuery(req.FullCodePath),
		withVersionQuery(req.Version),
		withIncludeTreeQuery(req.IncludeTree),
	))

	// 使用完整 URL 调用（不需要额外 header，但保留 ctx 用于超时控制）。
	result, err := CallAPIWithURL[*dto.HubDirectoryDetailDetailResp](ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, err
	}
	return result.Data, nil
}

// IncrementDownloadCountOnHost 在指定 Hub 主机上增加目录的下载次数（复制成功后调用）
// host 如 hub.example.com 或 http://hub.example.com
func IncrementDownloadCountOnHost(ctx context.Context, host, fullCodePath string) error {
	fullURL := buildHubAPIURL(host, "/hub/api/v1/directories/increment_download", nil)
	body := map[string]string{"full_code_path": fullCodePath}
	_, err := CallAPIWithURL[map[string]interface{}](ctx, http.MethodPost, fullURL, body)
	return err
}
