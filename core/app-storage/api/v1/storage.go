package v1

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"

	"github.com/ai-agent-os/ai-agent-os/core/app-storage/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-storage/service"
	"github.com/ai-agent-os/ai-agent-os/core/app-storage/storage"
	"github.com/ai-agent-os/ai-agent-os/dto"
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
// @Description 获取文件上传的预签名 URL，文件将按函数路径分类存储。如果未提供 router，将使用默认路由：/{username}/default
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

	// 获取当前登录用户的用户名
	username := contextx.GetRequestUser(c)

	// 如果未提供 Router，使用默认路由：/{username}/default
	router := req.Router
	if router == "" {
		if username == "" {
			response.FailWithMessage(c, "未提供路由且无法获取用户信息")
			return
		}
		router = fmt.Sprintf("%s/default", username)
		logger.Infof(c, "Router not provided, using default router: %s", router)
	}

	// 设置默认上传来源
	uploadSource := getDefaultUploadSource(req.UploadSource)

	// 将 gin.Context 转换为标准 context.Context
	ctx := contextx.ToContext(c)

	// 生成上传凭证
	creds, key, expire, err := s.storageService.GenerateUploadToken(ctx, router, req.FileName, req.ContentType, req.FileSize, uploadSource)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	// 获取 CDN 域名
	cdnDomain := s.storageService.GetCDNDomain()

	// 获取存储引擎类型
	storageType := s.storageService.GetStorageType()

	// 构建预期的下载URL
	downloadURL, serverDownloadURL, _, err := s.storageService.GetFileURLs(ctx, key)
	if err != nil {
		logger.Errorf(c, "Failed to generate download URLs: %v", err)
		// 下载URL生成失败不影响上传，设置为空
		downloadURL = ""
		serverDownloadURL = ""
	}

	// 构建响应
	resp = buildUploadTokenResponse(creds, key, expire, cdnDomain, storageType, downloadURL, serverDownloadURL, username)
	resp.Bucket = s.storageService.GetBucketName()

	response.OkWithData(c, resp)
}

// BatchGetUploadToken 批量获取上传凭证
// @Summary 批量获取上传凭证
// @Description 批量获取多个文件的上传凭证，支持多种存储方式（presigned_url/form_upload/sdk_upload）。如果某个文件未提供 router，将使用默认路由：/{username}/default
// @Tags 存储管理
// @Accept json
// @Produce json
// @Param request body dto.BatchGetUploadTokenReq true "批量获取上传凭证请求"
// @Success 200 {object} dto.BatchGetUploadTokenResp "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/storage/batch_upload_token [post]
func (s *Storage) BatchGetUploadToken(c *gin.Context) {
	var req dto.BatchGetUploadTokenReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	// 获取当前登录用户的用户名
	username := contextx.GetRequestUser(c)
	if username == "" {
		response.FailWithMessage(c, "无法获取用户信息")
		return
	}

	// 设置默认上传来源
	defaultUploadSource := getDefaultUploadSource(req.UploadSource)

	// 将 gin.Context 转换为标准 context.Context
	ctx := contextx.ToContext(c)

	// 批量生成上传凭证
	tokens := make([]dto.GetUploadTokenResp, 0, len(req.Files))
	for _, fileReq := range req.Files {
		// 优先使用文件项中的上传来源，如果没有则使用顶层的
		uploadSource := getDefaultUploadSource(fileReq.UploadSource)
		if uploadSource == "" {
			uploadSource = defaultUploadSource
		}

		// 如果未提供 Router，使用默认路由：/{username}/default
		router := fileReq.Router
		if router == "" {
			router = fmt.Sprintf("%s/default", username)
			logger.Infof(c, "Router not provided for file %s, using default router: %s", fileReq.FileName, router)
		}

		// 生成上传凭证
		creds, key, expire, err := s.storageService.GenerateUploadToken(ctx, router, fileReq.FileName, fileReq.ContentType, fileReq.FileSize, uploadSource)
		if err != nil {
			// 单个文件失败，记录错误但继续处理其他文件
			logger.Errorf(c, "Failed to generate upload token for file %s: %v", fileReq.FileName, err)
			continue
		}

		// 获取 CDN 域名和存储引擎类型
		cdnDomain := s.storageService.GetCDNDomain()
		storageType := s.storageService.GetStorageType()

		// 构建预期的下载URL
		downloadURL, serverDownloadURL, _, err := s.storageService.GetFileURLs(ctx, key)
		if err != nil {
			logger.Errorf(c, "Failed to generate download URLs for key %s: %v", key, err)
			// 下载URL生成失败不影响上传，设置为空
			downloadURL = ""
			serverDownloadURL = ""
		}

		// 构建响应
		token := buildUploadTokenResponse(creds, key, expire, cdnDomain, storageType, downloadURL, serverDownloadURL, username)
		token.Bucket = s.storageService.GetBucketName()
		tokens = append(tokens, *token)
	}

	response.OkWithData(c, dto.BatchGetUploadTokenResp{
		Tokens: tokens,
	})
}

// UploadComplete 上传完成通知
// @Summary 上传完成通知
// @Description 前端上传完成后，通知后端创建上传记录（仅在上传成功时记录）
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

	// 将 gin.Context 转换为标准 context.Context
	ctx := contextx.ToContext(c)

	// 只有上传成功时才创建记录
	var downloadURL string
	var serverDownloadURL string
	var expireStr string
	if req.Success {
		// 获取用户信息并创建上传记录
		requestUser := contextx.GetRequestUser(c)
		if err := createUploadRecord(
			s.storageService,
			ctx,
			req.Key,
			req.Router,
			req.FileName,
			req.FileSize,
			req.ContentType,
			req.Hash,
			requestUser,
		); err != nil {
			logger.Errorf(c, "Failed to record upload to database: %v (file_key: %s)", err, req.Key)
			// 不影响响应，只记录错误（文件已上传到MinIO，只是数据库记录失败）
		}

		// 构建下载URL
		var expire time.Time
		var err error
		downloadURL, serverDownloadURL, expire, err = s.storageService.GetFileURLs(ctx, req.Key)
		if err != nil {
			logger.Errorf(c, "Failed to generate download URLs: %v", err)
			downloadURL = ""
			serverDownloadURL = ""
		}

		// 获取过期时间
		if !expire.IsZero() {
			expireStr = expire.Format(storage.TimeFormat)
		} else {
			// 默认使用标准过期时间
			expire = time.Now().Add(storage.DefaultDownloadURLExpiry)
			expireStr = expire.Format(storage.TimeFormat)
		}

		logger.Infof(c, "Upload complete: key=%s, success=true", req.Key)
	} else {
		// 上传失败，不记录数据库，只记录日志
		logger.Warnf(c, "Upload failed: key=%s, error=%s", req.Key, req.Error)
	}

	// 构建响应（包含下载 URL）
	resp := &dto.UploadCompleteResp{
		Message:           "上传完成通知已处理",
		DownloadURL:       downloadURL,
		ServerDownloadURL: serverDownloadURL,
		Expire:            expireStr,
	}

	response.OkWithData(c, resp)
}

// BatchUploadComplete 批量上传完成通知
// @Summary 批量上传完成通知
// @Description 批量通知后端创建上传记录（仅在上传成功时记录）
// @Tags 存储管理
// @Accept json
// @Produce json
// @Param request body dto.BatchUploadCompleteReq true "批量上传完成请求"
// @Success 200 {object} dto.BatchUploadCompleteResp "通知成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/storage/batch_upload_complete [post]
func (s *Storage) BatchUploadComplete(c *gin.Context) {
	var req dto.BatchUploadCompleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	// 将 gin.Context 转换为标准 context.Context
	ctx := contextx.ToContext(c)

	// 获取用户信息
	requestUser := contextx.GetRequestUser(c)

	// 批量创建上传记录（仅成功时）
	results := make([]dto.BatchUploadCompleteResult, 0, len(req.Items))
	for _, item := range req.Items {
		if item.Success {
			// 只有上传成功时才创建记录
			if err := createUploadRecord(
				s.storageService,
				ctx,
				item.Key,
				item.Router,
				item.FileName,
				item.FileSize,
				item.ContentType,
				item.Hash,
				requestUser,
			); err != nil {
				logger.Errorf(c, "Failed to record upload for key %s: %v", item.Key, err)
				results = append(results, dto.BatchUploadCompleteResult{
					Key:    item.Key,
					Status: "failed",
					Error:  fmt.Sprintf("记录失败: %v", err),
				})
				continue
			}

			// 构建下载URL
			downloadURL, serverDownloadURL, _, err := s.storageService.GetFileURLs(ctx, item.Key)
			if err != nil {
				logger.Errorf(c, "Failed to generate download URLs for key %s: %v", item.Key, err)
				downloadURL = ""
				serverDownloadURL = ""
			}

			logger.Infof(c, "Upload complete: key=%s, success=true", item.Key)

			results = append(results, dto.BatchUploadCompleteResult{
				Key:               item.Key,
				Status:            "completed",
				DownloadURL:       downloadURL,       // ✨ 外部访问的下载地址（前端使用）
				ServerDownloadURL: serverDownloadURL, // ✨ 内部访问的下载地址（服务端使用）
				Hash:              item.Hash,         // ✨ 文件hash（用于文件缓存去重）
			})
		} else {
			// 上传失败，不记录数据库，只记录日志
			logger.Warnf(c, "Upload failed: key=%s, error=%s", item.Key, item.Error)
			results = append(results, dto.BatchUploadCompleteResult{
				Key:    item.Key,
				Status: "failed",
				Error:  item.Error,
			})
		}
	}

	response.OkWithData(c, dto.BatchUploadCompleteResp{
		Results: results,
	})
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
	// 使用 *key 匹配时，需要去掉前导斜杠
	key := c.Param("key")
	key = trimLeadingSlash(key)
	if key == "" {
		response.FailWithMessage(c, "文件 Key 不能为空")
		return
	}

	// 将 gin.Context 转换为标准 context.Context
	ctx := contextx.ToContext(c)

	// 获取文件信息
	info, err := s.storageService.GetFileInfo(ctx, key)
	if err != nil {
		response.FailWithMessage(c, "文件不存在或无法访问")
		return
	}

	// 记录下载（如果启用）
	requestUser := contextx.GetRequestUser(c)

	// 获取客户端 IP 和 User-Agent（规范化IP地址）
	ipAddress := normalizeIP(c.ClientIP())
	userAgent := c.GetHeader("User-Agent")

	// 创建下载记录（只记录 username，不记录 user_id）
	var usernameValue *string
	if requestUser != "" {
		usernameValue = &requestUser
	}

	downloadRecord := &model.FileDownload{
		FileKey:   key,
		UserID:    nil,
		Username:  usernameValue,
		IPAddress: ipAddress,
		UserAgent: userAgent,
	}

	// 异步记录下载（不阻塞响应）
	go func() {
		// 使用新的 context，避免使用可能已取消的请求 context
		ctx := context.Background()
		if err := s.storageService.RecordDownload(ctx, downloadRecord); err != nil {
			logger.Errorf(c, "Failed to record download: %v", err)
			// 不影响下载流程，只记录错误
		}
	}()

	// 直接代理文件下载（流式传输）
	bucket := s.storageService.GetBucketName()
	reader, err := s.storageService.GetStorage().DownloadObject(ctx, bucket, key)
	if err != nil {
		logger.Errorf(c, "Failed to download file: %v", err)
		response.FailWithMessage(c, "下载文件失败")
		return
	}
	defer reader.Close()

	// 设置响应头
	fileName := key
	if lastSlash := strings.LastIndex(key, "/"); lastSlash != -1 {
		fileName = key[lastSlash+1:]
	}

	c.Header("Content-Type", info.ContentType)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	c.Header("Content-Length", fmt.Sprintf("%d", info.Size))

	// 流式传输文件内容
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
	// 使用 *key 匹配时，需要去掉前导斜杠
	key := c.Param("key")
	key = trimLeadingSlash(key)
	if key == "" {
		response.FailWithMessage(c, "文件 Key 不能为空")
		return
	}

	// 将 gin.Context 转换为标准 context.Context
	ctx := contextx.ToContext(c)

	err := s.storageService.DeleteFile(ctx, key)
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
	// 使用 *key 匹配时，需要去掉前导斜杠
	key := c.Param("key")
	key = trimLeadingSlash(key)
	if key == "" {
		response.FailWithMessage(c, "文件 Key 不能为空")
		return
	}

	// 将 gin.Context 转换为标准 context.Context
	ctx := contextx.ToContext(c)

	info, err := s.storageService.GetFileInfo(ctx, key)
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

	// 将 gin.Context 转换为标准 context.Context
	ctx := contextx.ToContext(c)

	fileCount, totalSize, err := s.storageService.GetStorageStats(ctx, req.Router)
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

	// 将 gin.Context 转换为标准 context.Context
	ctx := contextx.ToContext(c)

	files, err := s.storageService.ListFilesByRouter(ctx, req.Router)
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

	// 将 gin.Context 转换为标准 context.Context
	ctx := contextx.ToContext(c)

	deletedCount, err := s.storageService.DeleteFilesByRouter(ctx, req.Router)
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

// trimLeadingSlash 移除前导斜杠（用于 *key 路由参数）
// 注意：此函数与 service/storage_service.go 中的 trimLeadingSlash 功能相同，但保留在各自包中以避免循环依赖
func trimLeadingSlash(s string) string {
	for len(s) > 0 && s[0] == '/' {
		s = s[1:]
	}
	return s
}

// normalizeIP 规范化IP地址（将IPv6的::1转换为127.0.0.1）
func normalizeIP(ip string) string {
	if ip == storage.IPv6Loopback {
		return storage.IPv4Loopback
	}
	// 尝试解析IP地址，如果是IPv6映射的IPv4地址，转换为IPv4
	parsedIP := net.ParseIP(ip)
	if parsedIP != nil {
		// 如果是IPv6映射的IPv4地址（::ffff:127.0.0.1），转换为IPv4
		if ipv4 := parsedIP.To4(); ipv4 != nil {
			return ipv4.String()
		}
	}
	return ip
}

// formatSize 格式化文件大小
func formatSize(size int64) string {
	if size < storage.BytesPerKB {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(storage.BytesPerKB), 0
	for n := size / storage.BytesPerKB; n >= storage.BytesPerKB; n /= storage.BytesPerKB {
		div *= storage.BytesPerKB
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}
