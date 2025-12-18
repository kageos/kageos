package apicall

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/serviceconfig"
)

type ApiResult[T any] struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data T      `json:"data"`
}

// Header 后续所有的请求都走网关，这样的话，网关过来的请求都是有用户信息的，同样也有trace_id和token，我们在下游想要再调用另外的服务需要经过网关，这样的话我们是需要拿着这个token再调用的
type Header struct {
	TraceID     string `json:"trace_id"`
	RequestUser string `json:"request_user"`
	Token       string `json:"token"` // ✨ 使用前端透传过来的token
}

// httpClient 通用HTTP客户端（复用连接，提高性能）
var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

// callAPI 通用API调用方法
// method: HTTP方法（POST、GET等）
// path: API路径（如 "/storage/api/v1/upload_token"）
// header: 请求头信息（包含token、trace_id等）
// reqBody: 请求体（会被序列化为JSON）
// 返回: ApiResult[T] 结构，包含code、msg、data字段
func callAPI[T any](method, path string, header *Header, reqBody interface{}) (*ApiResult[T], error) {
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
	
	// 4. 创建HTTP请求
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	
	// 5. 设置请求头
	req.Header.Set("Content-Type", "application/json")

	// ✨ 使用Token方式（透传前端传过来的token）
	if header != nil && header.Token != "" {
		req.Header.Set("X-Token", header.Token)
	}

	// 设置追踪ID（使用统一的header key）
	if header != nil && header.TraceID != "" {
		req.Header.Set("X-Trace-Id", header.TraceID)
	}

	// ✨ 设置请求用户（用于区分前端请求和容器内SDK请求）
	if header != nil && header.RequestUser != "" {
		req.Header.Set("X-Request-User", header.RequestUser)
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
func GetUploadToken(header *Header, req *dto.GetUploadTokenReq) (*dto.GetUploadTokenResp, error) {
	result, err := callAPI[dto.GetUploadTokenResp](http.MethodPost, "/storage/api/v1/upload_token", header, req)
	if err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// BatchGetUploadToken 批量获取上传凭证
func BatchGetUploadToken(header *Header, req *dto.BatchGetUploadTokenReq) (*dto.BatchGetUploadTokenResp, error) {
	result, err := callAPI[dto.BatchGetUploadTokenResp](http.MethodPost, "/storage/api/v1/batch_upload_token", header, req)
	if err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// UploadComplete 通知上传完成（单个）
func UploadComplete(header *Header, req *dto.UploadCompleteReq) (*dto.UploadCompleteResp, error) {
	result, err := callAPI[dto.UploadCompleteResp](http.MethodPost, "/storage/api/v1/upload_complete", header, req)
	if err != nil {
		return nil, err
	}
	return &result.Data, nil
}

// BatchUploadComplete 批量通知上传完成
func BatchUploadComplete(header *Header, req *dto.BatchUploadCompleteReq) (*dto.BatchUploadCompleteResp, error) {
	result, err := callAPI[dto.BatchUploadCompleteResp](http.MethodPost, "/storage/api/v1/batch_upload_complete", header, req)
	if err != nil {
		return nil, err
	}
	return &result.Data, nil
}
