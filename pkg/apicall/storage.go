package apicall

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/serviceconfig"
)

type ApiResult[T any] struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data T      `json:"data"`
}

// Header 已废弃，请使用 context.Context 替代
// 保留此类型仅用于向后兼容，新代码请使用 WithCtx 方法
// Deprecated: 使用 context.Context 替代，通过 contextx 包提取 token、trace_id、request_user
type Header struct {
	TraceID     string `json:"trace_id"`
	RequestUser string `json:"request_user"`
	Token       string `json:"token"`
}

// httpClient 通用HTTP客户端（复用连接，提高性能）
var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

// callAPI 通用API调用方法
// method: HTTP方法（POST、GET等）
// path: API路径（如 "/storage/api/v1/upload_token"）
// ctx: 上下文（从 ctx 中提取 token、trace_id、request_user）
// reqBody: 请求体（会被序列化为JSON）
// 返回: ApiResult[T] 结构，包含code、msg、data字段
func callAPI[T any](ctx context.Context, method, path string, reqBody interface{}) (*ApiResult[T], error) {
	// 如果 ctx 为 nil，使用 Background
	if ctx == nil {
		ctx = context.Background()
	}

	// 1. 获取Gateway URL（方式1：从环境变量获取）
	gatewayURL := serviceconfig.GetGatewayURL()
	
	// 2. 构建完整URL
	url := gatewayURL + path
	
	// 3. 序列化请求体
	var bodyReader io.Reader
	if reqBody != nil {
		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("序列化请求体失败: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}
	
	// 4. 创建HTTP请求（使用 ctx，支持超时和取消）
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	
	// 5. 设置请求头
	req.Header.Set("Content-Type", "application/json")

	// ✨ 从 ctx 中提取 Token（透传前端传过来的token）
	if token := contextx.GetToken(ctx); token != "" {
		req.Header.Set("X-Token", token)
	}

	// ✨ 从 ctx 中提取追踪ID（使用统一的header key）
	if traceID := contextx.GetTraceId(ctx); traceID != "" {
		req.Header.Set("X-Trace-Id", traceID)
	}

	// ✨ 从 ctx 中提取请求用户（用于区分前端请求和容器内SDK请求）
	if requestUser := contextx.GetRequestUser(ctx); requestUser != "" {
		req.Header.Set("X-Request-User", requestUser)
	}
	
	// 6. 发送请求
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()
	
	// 7. 读取响应体
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}
	
	// 8. 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP错误: %d %s, 响应: %s", resp.StatusCode, resp.Status, string(bodyBytes))
	}
	
	// 9. 解析响应（直接使用ApiResult[T]泛型）
	var result ApiResult[T]
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w, 响应内容: %s", err, string(bodyBytes))
	}
	
	// 10. 检查业务错误（code != 0）
	if result.Code != 0 {
		return &result, fmt.Errorf("业务错误 [%d]: %s", result.Code, result.Msg)
	}
	
	// 11. 返回成功结果
	return &result, nil
}

// GetUploadToken 获取上传凭证（单个）
func GetUploadToken(ctx context.Context, req *dto.GetUploadTokenReq) (*dto.GetUploadTokenResp, error) {
	return PostAPI[*dto.GetUploadTokenReq, *dto.GetUploadTokenResp](ctx, "/storage/api/v1/upload_token", req)
}

// BatchGetUploadToken 批量获取上传凭证
func BatchGetUploadToken(ctx context.Context, req *dto.BatchGetUploadTokenReq) (*dto.BatchGetUploadTokenResp, error) {
	return PostAPI[*dto.BatchGetUploadTokenReq, *dto.BatchGetUploadTokenResp](ctx, "/storage/api/v1/batch_upload_token", req)
}

// UploadComplete 通知上传完成（单个）
func UploadComplete(ctx context.Context, req *dto.UploadCompleteReq) (*dto.UploadCompleteResp, error) {
	return PostAPI[*dto.UploadCompleteReq, *dto.UploadCompleteResp](ctx, "/storage/api/v1/upload_complete", req)
}

// BatchUploadComplete 批量通知上传完成
func BatchUploadComplete(ctx context.Context, req *dto.BatchUploadCompleteReq) (*dto.BatchUploadCompleteResp, error) {
	return PostAPI[*dto.BatchUploadCompleteReq, *dto.BatchUploadCompleteResp](ctx, "/storage/api/v1/batch_upload_complete", req)
}
