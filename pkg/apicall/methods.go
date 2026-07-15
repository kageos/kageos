package apicall

import (
	"context"
	"net/http"
	"net/url"
)

// GetAPI 发送 GET 请求（无请求体，参数通过 URL query string 传递）
// T: 响应类型（指针类型，如 *dto.SearchFunctionsResp）
// ctx: 上下文（从 ctx 中提取 token、trace_id、request_user）
// path: API路径（如 "/workspace/api/v1/service_tree/123" 或已包含 query params 的完整路径）
// queryParams: URL查询参数（可选，nil 表示无查询参数或 path 已包含 query params）
// 返回: T（指针类型）
func GetAPI[T any](ctx context.Context, path string, queryParams url.Values) (T, error) {
	fullPath := buildPathWithQuery(path, queryParams)

	result, err := callAPI[T](ctx, http.MethodGet, fullPath, nil)
	if err != nil {
		var zero T
		return zero, err
	}
	return result.Data, nil
}

// PostAPI 发送 POST 请求（带请求体）
// TReq: 请求类型（可以是值类型或指针类型）
// TResp: 响应类型（指针类型，如 *dto.AddFunctionsResp）
// ctx: 上下文（从 ctx 中提取 token、trace_id、request_user）
// path: API路径（如 "/workspace/api/v1/service_tree/add_functions"）
// req: 请求体（会被序列化为JSON）
// 返回: TResp（指针类型）
func PostAPI[TReq, TResp any](ctx context.Context, path string, req TReq) (TResp, error) {
	result, err := callAPI[TResp](ctx, http.MethodPost, path, req)
	if err != nil {
		var zero TResp
		return zero, err
	}
	return result.Data, nil
}

// PutAPI 发送 PUT 请求（带请求体）
// TReq: 请求类型（可以是值类型或指针类型）
// TResp: 响应类型（指针类型，如 *dto.UpdateWorkspaceResp）
// ctx: 上下文（从 ctx 中提取 token、trace_id、request_user）
// path: API路径（如 "/workspace/api/v1/service_tree/update"）
// req: 请求体（会被序列化为JSON）
// 返回: TResp（指针类型）
func PutAPI[TReq, TResp any](ctx context.Context, path string, req TReq) (TResp, error) {
	result, err := callAPI[TResp](ctx, http.MethodPut, path, req)
	if err != nil {
		var zero TResp
		return zero, err
	}
	return result.Data, nil
}

// DeleteBodyAPI 发送 DELETE 请求（带请求体）
// 用于需要 body 的接口，例如 Table 批量删除：{"ids":[1,2,3]}。
func DeleteBodyAPI[TReq, TResp any](ctx context.Context, path string, req TReq) (TResp, error) {
	result, err := callAPI[TResp](ctx, http.MethodDelete, path, req)
	if err != nil {
		var zero TResp
		return zero, err
	}
	return result.Data, nil
}

func buildPathWithQuery(path string, queryParams url.Values) string {
	if len(queryParams) == 0 {
		return path
	}
	if encoded := queryParams.Encode(); encoded != "" {
		separator := "?"
		if len(path) > 0 && path[len(path)-1] == '?' {
			separator = ""
		} else if parsed, err := url.Parse(path); err == nil && parsed.RawQuery != "" {
			separator = "&"
		}
		return path + separator + encoded
	}
	return path
}
