package v1

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/app-storage/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-storage/service"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/response"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/gin-gonic/gin"
)

// Storage 存储相关API
type Storage struct {
	storageService *service.StorageService
}

// NewStorage 创建存储API（依赖注入）
func NewStorage(storageService *service.StorageService) *Storage {
	return &Storage{
		storageService: storageService,
	}
}

// GetUploadToken 获取上传凭证
// @Summary 获取上传凭证
// @Description 获取文件上传的预签名 URL，文件将按函数路径分类存储
// @Tags 存储管理
// @Accept json
// @Produce json
// @Param request body dto.GetUploadTokenReq true "获取上传凭证请求"
// @Success 200 {object} dto.GetUploadTokenResp "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/storage/upload_token [post]
func (s *Storage) GetUploadToken(c *gin.Context) {
	var req dto.GetUploadTokenReq
	var resp *dto.GetUploadTokenResp
	var err error
	defer func() {
		logger.Infof(c, "GetUploadToken req:%+v resp:%+v err:%v", req, resp, err)
	}()

	// 绑定请求参数
	if err = c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	// ✅ 获取用户信息（使用与 app-server 一致的辅助函数）
	// ✅ username 是不可变的，因此无需记录 user_id
	requestUser := contextx.GetRequestUser(c) // ✅ 使用与 app-server 一致的辅助函数

	// ✅ 从 router 中提取 tenant（第一段）
	// 例如：luobei/test88888/crm/crm_ticket → tenant = luobei
	tenant := extractTenantFromRouter(req.Router)

	// 生成上传凭证（包含函数路径）
	creds, key, expire, err := s.storageService.GenerateUploadToken(c.Request.Context(), req.Router, req.FileName, req.ContentType, req.FileSize)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	// 记录上传记录（预记录，状态为 pending）
	// ✅ 只记录 username，不记录 user_id（username 不可变）
	uploadRecord := &model.FileUpload{
		FileKey:     key,
		Router:      req.Router,
		FileName:    req.FileName,
		FileSize:    req.FileSize,
		ContentType: req.ContentType,
		Hash:        req.Hash,
		UserID:      nil, // ✅ 不记录 user_id，username 不可变
		Username:    requestUser,
		Tenant:      tenant,
		Status:      "pending",
	}
	if err := s.storageService.RecordUpload(c.Request.Context(), uploadRecord); err != nil {
		logger.Errorf(c, "Failed to record upload: %v", err)
		// 不影响上传流程，只记录错误
	}

	// 获取 CDN 域名
	cdnDomain := s.storageService.GetCDNDomain()
	
	// ✅ 获取存储引擎类型（用于记录文件存储位置）
	storageType := s.storageService.GetStorageType()

	// 构建响应（根据上传方式返回不同字段）
	resp = &dto.GetUploadTokenResp{
		Key:         key,
		Bucket:      s.storageService.GetBucketName(),
		Expire:      expire.Format("2006-01-02 15:04:05"),
		Method:      dto.UploadMethod(creds.Method),
		Storage:     storageType, // ✨ 存储引擎类型（minio/qiniu/tencentcos/aliyunoss/awss3）
		URL:         creds.URL,
		Headers:     creds.Headers,
		UploadHost:  creds.UploadHost,   // ✨ 上传目标 host（用于 CORS、进度监听）
		UploadDomain: creds.UploadDomain, // ✨ 上传完整域名（用于日志、调试）
		FormData:    creds.FormData,
		PostURL:     creds.PostURL,
		SDKConfig:   creds.SDKConfig,
		CDNDomain:   cdnDomain, // ✨ CDN 域名（用于下载访问）
	}

	response.OkWithData(c, resp)
}

// UploadComplete 上传完成通知
// @Summary 上传完成通知
// @Description 前端上传完成后，通知后端更新上传记录状态
// @Tags 存储管理
// @Accept json
// @Produce json
// @Param request body dto.UploadCompleteReq true "上传完成请求"
// @Success 200 {object} dto.UploadCompleteResp "通知成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/storage/upload_complete [post]
func (s *Storage) UploadComplete(c *gin.Context) {
	var req dto.UploadCompleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	// 更新上传记录状态
	status := "completed"
	if !req.Success {
		status = "failed"
	}

	err := s.storageService.UpdateUploadStatus(c.Request.Context(), req.Key, status)
	if err != nil {
		logger.Errorf(c, "Failed to update upload status: %v", err)
		response.FailWithMessage(c, "更新上传状态失败")
		return
	}

	logger.Infof(c, "Upload complete: key=%s, success=%v", req.Key, req.Success)

	// ✅ 如果上传成功，返回通过 storage 服务代理的下载 URL（而不是 MinIO 的直接 URL）
	var downloadURL string
	var expireStr string
	if req.Success {
		// ✅ 生成通过 storage 服务代理的下载 URL
		// 格式：http://{host}/api/v1/storage/download/{key}
		// 这样前端可以直接访问，不需要直接访问 MinIO
		scheme := "http"
		if c.Request.TLS != nil {
			scheme = "https"
		}
		host := c.Request.Host
		if host == "" {
			// 降级：使用配置的端口（通过反射或其他方式获取，这里简化处理）
			// 实际上 Request.Host 通常不会为空，但为了安全起见保留降级逻辑
			host = "localhost:8083" // 默认端口
		}
		
		// ✅ 构建代理下载 URL
		downloadURL = fmt.Sprintf("%s://%s/api/v1/storage/download/%s", scheme, host, url.PathEscape(req.Key))
		
		// 获取过期时间（7天）
		expire := time.Now().Add(7 * 24 * time.Hour)
		expireStr = expire.Format("2006-01-02 15:04:05")
	}

	// 构建响应（包含下载 URL）
	resp := &dto.UploadCompleteResp{
		Message:     "上传状态已更新",
		DownloadURL: downloadURL,
		Expire:      expireStr,
	}

	response.OkWithData(c, resp)
}

// GetFileURL 获取文件（直接代理下载，返回简洁 URL）
// @Summary 下载文件
// @Description 直接代理下载文件，返回文件流（无需复杂的预签名 URL 参数）
// @Tags 存储管理
// @Accept json
// @Produce application/octet-stream
// @Param key path string true "文件 Key"
// @Success 200 {file} file "文件内容"
// @Failure 400 {string} string "请求参数错误"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/storage/download/{key} [get]
func (s *Storage) GetFileURL(c *gin.Context) {
	// ✅ 使用 *key 匹配时，需要去掉前导斜杠
	key := c.Param("key")
	key = trimLeadingSlash(key)
	if key == "" {
		response.FailWithMessage(c, "文件 Key 不能为空")
		return
	}

	// ✅ 获取文件信息
	info, err := s.storageService.GetFileInfo(c.Request.Context(), key)
	if err != nil {
		response.FailWithMessage(c, "文件不存在或无法访问")
		return
	}

	// ✅ 记录下载（如果启用）
	// 获取用户信息（使用与 app-server 一致的辅助函数）
	// ✅ username 是不可变的，因此无需记录 user_id
	requestUser := contextx.GetRequestUser(c) // ✅ 使用与 app-server 一致的辅助函数
	
	// 获取客户端 IP 和 User-Agent
	ipAddress := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")
	
	// 创建下载记录（只记录 username，不记录 user_id）
	var usernameValue *string
	if requestUser != "" {
		usernameValue = &requestUser
	}
	
	downloadRecord := &model.FileDownload{
		FileKey:     key,
		UserID:      nil, // ✅ 不记录 user_id，username 不可变
		Username:    usernameValue,
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
	}
	
	// 异步记录下载（不阻塞响应）
	go func() {
		// ✅ 使用新的 context，避免使用可能已取消的请求 context
		ctx := context.Background()
		if err := s.storageService.RecordDownload(ctx, downloadRecord); err != nil {
			logger.Errorf(c, "Failed to record download: %v", err)
			// 不影响下载流程，只记录错误
		}
	}()

	// ✅ 直接代理文件下载（流式传输）
	bucket := s.storageService.GetBucketName()
	reader, err := s.storageService.GetStorage().DownloadObject(c.Request.Context(), bucket, key)
	if err != nil {
		logger.Errorf(c, "Failed to download file: %v", err)
		response.FailWithMessage(c, "下载文件失败")
		return
	}
	defer reader.Close()

	// ✅ 设置响应头
	fileName := key
	if lastSlash := strings.LastIndex(key, "/"); lastSlash != -1 {
		fileName = key[lastSlash+1:]
	}
	
	c.Header("Content-Type", info.ContentType)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	c.Header("Content-Length", fmt.Sprintf("%d", info.Size))
	
	// ✅ 流式传输文件内容
	c.DataFromReader(http.StatusOK, info.Size, info.ContentType, reader, nil)
}

// DeleteFile 删除文件
// @Summary 删除文件
// @Description 删除存储的文件
// @Tags 存储管理
// @Accept json
// @Produce json
// @Param key path string true "文件 Key"
// @Success 200 {string} string "删除成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/storage/files/{key} [delete]
func (s *Storage) DeleteFile(c *gin.Context) {
	// ✅ 使用 *key 匹配时，需要去掉前导斜杠
	key := c.Param("key")
	key = trimLeadingSlash(key)
	if key == "" {
		response.FailWithMessage(c, "文件 Key 不能为空")
		return
	}

	err := s.storageService.DeleteFile(c.Request.Context(), key)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithMessage(c, "文件删除成功")
}

// GetFileInfo 获取文件信息
// @Summary 获取文件信息
// @Description 获取文件的元数据信息
// @Tags 存储管理
// @Accept json
// @Produce json
// @Param key path string true "文件 Key"
// @Success 200 {object} dto.GetFileInfoResp "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/storage/files/{key}/info [get]
func (s *Storage) GetFileInfo(c *gin.Context) {
	// ✅ 使用 *key 匹配时，需要去掉前导斜杠
	key := c.Param("key")
	key = trimLeadingSlash(key)
	if key == "" {
		response.FailWithMessage(c, "文件 Key 不能为空")
		return
	}

	info, err := s.storageService.GetFileInfo(c.Request.Context(), key)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	resp := &dto.GetFileInfoResp{
		Key:          info.Key,
		Size:         info.Size,
		ContentType:  info.ContentType,
		ETag:         info.ETag,
		LastModified: info.LastModified.Format(http.TimeFormat),
	}

	response.OkWithData(c, resp)
}

// GetStorageStats 获取存储统计信息
// @Summary 获取存储统计信息
// @Description 获取某个函数路径下的文件数量和总大小
// @Tags 存储管理
// @Accept json
// @Produce json
// @Param router query string true "函数路径，例如：luobei/test88888/tools/cashier_desk"
// @Success 200 {object} dto.GetStorageStatsResp "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/storage/stats [get]
func (s *Storage) GetStorageStats(c *gin.Context) {
	var req dto.GetStorageStatsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	fileCount, totalSize, err := s.storageService.GetStorageStats(c.Request.Context(), req.Router)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	// 转换为人类可读的大小
	sizeHuman := formatSize(totalSize)

	resp := &dto.GetStorageStatsResp{
		Router:    req.Router,
		FileCount: fileCount,
		TotalSize: totalSize,
		SizeHuman: sizeHuman,
	}

	response.OkWithData(c, resp)
}

// ListFiles 列举文件
// @Summary 列举文件
// @Description 列举某个函数路径下的所有文件
// @Tags 存储管理
// @Accept json
// @Produce json
// @Param router query string true "函数路径"
// @Success 200 {object} dto.ListFilesResp "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/storage/files [get]
func (s *Storage) ListFiles(c *gin.Context) {
	var req dto.ListFilesReq
	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	files, err := s.storageService.ListFilesByRouter(c.Request.Context(), req.Router)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	resp := &dto.ListFilesResp{
		Router: req.Router,
		Files:  files,
		Count:  len(files),
	}

	response.OkWithData(c, resp)
}

// DeleteFilesByRouter 删除函数路径下的所有文件
// @Summary 删除函数路径下的所有文件
// @Description 批量删除某个函数路径下的所有文件（危险操作）
// @Tags 存储管理
// @Accept json
// @Produce json
// @Param request body dto.DeleteFilesByRouterReq true "删除请求"
// @Success 200 {object} dto.DeleteFilesByRouterResp "删除成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/storage/batch_delete [post]
func (s *Storage) DeleteFilesByRouter(c *gin.Context) {
	var req dto.DeleteFilesByRouterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	deletedCount, err := s.storageService.DeleteFilesByRouter(c.Request.Context(), req.Router)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	resp := &dto.DeleteFilesByRouterResp{
		Router:       req.Router,
		DeletedCount: deletedCount,
	}

	response.OkWithData(c, resp)
}

// ✅ extractTenantFromRouter 从 router 中提取 tenant
// 例如：luobei/test88888/crm/crm_ticket → tenant = luobei
func extractTenantFromRouter(router string) string {
	parts := strings.Split(router, "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// ✅ trimLeadingSlash 移除前导斜杠（用于 *key 路由参数）
func trimLeadingSlash(s string) string {
	for len(s) > 0 && s[0] == '/' {
		s = s[1:]
	}
	return s
}

// formatSize 格式化文件大小
func formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}


