package v1

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kageos/kageos/pkg/contextx"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/core/app-storage/model"
	"github.com/kageos/kageos/core/app-storage/service"
	"github.com/kageos/kageos/core/app-storage/storage"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/ginx/response"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/pkg/publicshare"
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

func (s *Storage) publicShareRequestUser(c *gin.Context, router string) (string, bool) {
	claims, err := publicshare.ValidateAnonymousToken(c.GetHeader(publicshare.AnonymousTokenHeader))
	if err != nil {
		response.FailWithMessage(c, "匿名访问凭证无效，请刷新页面后重试")
		return "", false
	}
	shareID := strings.TrimSpace(c.Param("share_id"))
	if shareID == "" {
		response.FailWithMessage(c, "分享链接无效")
		return "", false
	}
	tenantUser, app := parsePublicShareRouter(router)
	requestUser := publicshare.DeriveActorID(tenantUser, app, shareID, claims.SessionID)
	c.Request.Header.Set(contextx.RequestUserHeader, requestUser)
	return requestUser, true
}

func parsePublicShareRouter(router string) (string, string) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(router), "/"), "/")
	if len(parts) < 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

func isObjectKeyInRouter(key string, router string) bool {
	key = strings.Trim(strings.TrimSpace(key), "/")
	router = strings.Trim(strings.TrimSpace(router), "/")
	if key == "" || router == "" {
		return false
	}
	return key == router || strings.HasPrefix(key, router+"/")
}

func (s *Storage) PublicShareGetUploadToken(c *gin.Context) {
	var req dto.GetUploadTokenReq
	var resp *dto.GetUploadTokenResp
	var err error
	defer func() {
		logUploadTokenDebug(c, "PublicShareGetUploadToken", req, resp, err)
	}()

	if err = c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}
	router := strings.Trim(strings.TrimSpace(req.Router), "/")
	if router == "" {
		response.FailWithMessage(c, "公开上传必须提供函数路由")
		return
	}
	requestUser, ok := s.publicShareRequestUser(c, router)
	if !ok {
		return
	}

	uploadSource := getDefaultUploadSource(req.UploadSource)
	ctx := contextx.ToContext(c)
	presignHost := contextx.GetPresignHost(ctx)
	logger.Infof(c, "[PublicShareGetUploadToken] presign host for upload: X-Forwarded-Host=%q, Request.Host=%q => presignHost=%q", c.GetHeader("X-Forwarded-Host"), c.Request.Host, presignHost)

	bucket := req.Bucket
	if bucket == "" {
		bucket = s.storageService.GetBucketName()
	}
	var creds *storage.UploadCredentials
	var key string
	var expire time.Time
	if req.PreviewForKey != "" {
		if !isObjectKeyInRouter(req.PreviewForKey, router) {
			response.FailWithMessage(c, "预览文件路径不属于当前分享")
			return
		}
		creds, key, expire, err = s.storageService.GeneratePreviewUploadToken(ctx, bucket, req.PreviewForKey, req.FileName, req.ContentType, req.FileSize, uploadSource)
	} else {
		creds, key, expire, err = s.storageService.GenerateUploadToken(ctx, bucket, router, req.FileName, req.ContentType, req.FileSize, uploadSource)
	}
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	downloadURL, serverDownloadURL, _, err := s.storageService.GetFileURLsInBucket(ctx, bucket, key)
	if err != nil {
		logger.Errorf(c, "Failed to generate public share download URLs: %v", err)
		downloadURL = ""
		serverDownloadURL = ""
	}

	resp = buildUploadTokenResponse(
		creds,
		bucket,
		key,
		expire,
		s.storageService.GetCDNDomain(),
		s.storageService.GetStorageType(),
		downloadURL,
		serverDownloadURL,
		requestUser,
	)
	response.OkWithData(c, resp)
}

func (s *Storage) PublicShareUploadComplete(c *gin.Context) {
	var req dto.UploadCompleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}
	router := strings.Trim(strings.TrimSpace(req.Router), "/")
	if router == "" {
		response.FailWithMessage(c, "公开上传完成必须提供函数路由")
		return
	}
	if _, ok := s.publicShareRequestUser(c, router); !ok {
		return
	}
	if req.Success && !isObjectKeyInRouter(req.Key, router) {
		response.FailWithMessage(c, "文件路径不属于当前分享")
		return
	}

	ctx := contextx.ToContext(c)
	var downloadURL string
	var serverDownloadURL string
	var expireStr string
	var thumbnailURL string
	bucket := req.Bucket
	if bucket == "" {
		bucket = s.storageService.GetBucketName()
	}
	ref := s.storageService.BuildFileRef(bucket, req.Key)
	if req.Success {
		requestUser := contextx.GetRequestUser(c)
		if err := createUploadRecord(
			s.storageService,
			ctx,
			bucket,
			req.Key,
			router,
			req.FileName,
			req.Description,
			req.FileSize,
			req.ContentType,
			req.Hash,
			req.ThumbnailRef,
			req.PreviewKind,
			requestUser,
		); err != nil {
			logger.Errorf(c, "Failed to record public share upload to database: %v (file_key_len=%d)", err, len(req.Key))
		}

		var expire time.Time
		var err error
		downloadURL, serverDownloadURL, expire, err = s.storageService.GetFileURLsInBucket(ctx, bucket, req.Key)
		if err != nil {
			logger.Errorf(c, "Failed to generate public share download URLs: %v", err)
			downloadURL = ""
			serverDownloadURL = ""
		}
		thumbnailURL = s.browserURLForRef(ctx, req.ThumbnailRef)
		if !expire.IsZero() {
			expireStr = expire.Format(storage.TimeFormat)
		} else {
			expireStr = time.Now().Add(storage.DefaultDownloadURLExpiry).Format(storage.TimeFormat)
		}
	} else {
		logger.Warnf(c, "Public share upload failed: key=%s, error=%s", req.Key, req.Error)
	}

	response.OkWithData(c, &dto.UploadCompleteResp{
		Message:           "上传完成通知已处理",
		Key:               req.Key,
		Bucket:            bucket,
		Ref:               ref,
		FileName:          req.FileName,
		Description:       req.Description,
		FileSize:          req.FileSize,
		ContentType:       req.ContentType,
		Hash:              req.Hash,
		ThumbnailRef:      req.ThumbnailRef,
		ThumbnailURL:      thumbnailURL,
		PreviewKind:       req.PreviewKind,
		Storage:           s.storageService.GetStorageType(),
		DownloadURL:       downloadURL,
		ServerDownloadURL: serverDownloadURL,
		Expire:            expireStr,
	})
}

func (s *Storage) PublicShareBatchUploadComplete(c *gin.Context) {
	var req dto.BatchUploadCompleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	router := ""
	for _, item := range req.Items {
		if strings.TrimSpace(item.Router) != "" {
			router = strings.Trim(strings.TrimSpace(item.Router), "/")
			break
		}
	}
	if router == "" {
		response.FailWithMessage(c, "公开上传完成必须提供函数路由")
		return
	}
	if _, ok := s.publicShareRequestUser(c, router); !ok {
		return
	}

	ctx := contextx.ToContext(c)
	requestUser := contextx.GetRequestUser(c)
	results := make([]dto.BatchUploadCompleteResult, 0, len(req.Items))
	for _, item := range req.Items {
		itemRouter := strings.Trim(strings.TrimSpace(item.Router), "/")
		if itemRouter == "" {
			itemRouter = router
		}
		bucket := item.Bucket
		if bucket == "" {
			bucket = s.storageService.GetBucketName()
		}
		ref := s.storageService.BuildFileRef(bucket, item.Key)
		if item.Success {
			if !isObjectKeyInRouter(item.Key, itemRouter) {
				results = append(results, dto.BatchUploadCompleteResult{
					Key:    item.Key,
					Bucket: bucket,
					Ref:    ref,
					Status: "failed",
					Error:  "文件路径不属于当前分享",
				})
				continue
			}
			if err := createUploadRecord(
				s.storageService,
				ctx,
				bucket,
				item.Key,
				itemRouter,
				item.FileName,
				item.Description,
				item.FileSize,
				item.ContentType,
				item.Hash,
				item.ThumbnailRef,
				item.PreviewKind,
				requestUser,
			); err != nil {
				logger.Errorf(c, "Failed to record public share upload: key_len=%d, err=%v", len(item.Key), err)
				results = append(results, dto.BatchUploadCompleteResult{
					Key:    item.Key,
					Status: "failed",
					Error:  fmt.Sprintf("记录失败: %v", err),
				})
				continue
			}

			downloadURL, serverDownloadURL, _, err := s.storageService.GetFileURLsInBucket(ctx, bucket, item.Key)
			if err != nil {
				logger.Errorf(c, "Failed to generate public share download URLs: key_len=%d, err=%v", len(item.Key), err)
				downloadURL = ""
				serverDownloadURL = ""
			}
			results = append(results, dto.BatchUploadCompleteResult{
				Key:               item.Key,
				Bucket:            bucket,
				Ref:               ref,
				Status:            "completed",
				DownloadURL:       downloadURL,
				Description:       item.Description,
				ServerDownloadURL: serverDownloadURL,
				Hash:              item.Hash,
				ThumbnailRef:      item.ThumbnailRef,
				ThumbnailURL:      s.browserURLForRef(ctx, item.ThumbnailRef),
				PreviewKind:       item.PreviewKind,
			})
		} else {
			logger.Warnf(c, "Public share upload failed: key=%s, error=%s", item.Key, item.Error)
			results = append(results, dto.BatchUploadCompleteResult{
				Key:    item.Key,
				Bucket: bucket,
				Ref:    ref,
				Status: "failed",
				Error:  item.Error,
			})
		}
	}

	response.OkWithData(c, dto.BatchUploadCompleteResp{Results: results})
}

func (s *Storage) PublicShareResolveFileRefs(c *gin.Context) {
	var req dto.ResolveFileRefsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}
	if _, ok := s.publicShareRequestUser(c, ""); !ok {
		return
	}

	ctx := contextx.ToContext(c)
	files, err := s.storageService.ResolveFileRefs(ctx, req.Refs, req.Audience)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	response.OkWithData(c, dto.ResolveFileRefsResp{Files: files})
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
// @Router /storage/api/v1/upload_token [post]
func (s *Storage) GetUploadToken(c *gin.Context) {
	var req dto.GetUploadTokenReq
	var resp *dto.GetUploadTokenResp
	var err error
	defer func() {
		logUploadTokenDebug(c, "GetUploadToken", req, resp, err)
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
	// 诊断预签名 Host：用于排查 403（签名 Host 需与浏览器 PUT 的 Host 一致）
	presignHost := contextx.GetPresignHost(ctx)
	logger.Infof(c, "[GetUploadToken] presign host for upload: X-Forwarded-Host=%q, Request.Host=%q => presignHost=%q", c.GetHeader("X-Forwarded-Host"), c.Request.Host, presignHost)

	// 生成上传凭证
	bucket := req.Bucket
	if bucket == "" {
		bucket = s.storageService.GetBucketName()
	}
	var creds *storage.UploadCredentials
	var key string
	var expire time.Time
	if req.PreviewForKey != "" {
		creds, key, expire, err = s.storageService.GeneratePreviewUploadToken(ctx, bucket, req.PreviewForKey, req.FileName, req.ContentType, req.FileSize, uploadSource)
	} else {
		creds, key, expire, err = s.storageService.GenerateUploadToken(ctx, bucket, router, req.FileName, req.ContentType, req.FileSize, uploadSource)
	}
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	// 获取 CDN 域名
	cdnDomain := s.storageService.GetCDNDomain()

	// 获取存储引擎类型
	storageType := s.storageService.GetStorageType()

	// 构建预期的下载URL
	downloadURL, serverDownloadURL, _, err := s.storageService.GetFileURLsInBucket(ctx, bucket, key)
	if err != nil {
		logger.Errorf(c, "Failed to generate download URLs: %v", err)
		// 下载URL生成失败不影响上传，设置为空
		downloadURL = ""
		serverDownloadURL = ""
	}

	// 构建响应
	resp = buildUploadTokenResponse(creds, bucket, key, expire, cdnDomain, storageType, downloadURL, serverDownloadURL, username)

	// 线上排查 403：确认返回的 URL 的 host 与浏览器 PUT 时的 Host 一致
	if resp.UploadURL != "" {
		if u, e := url.Parse(resp.UploadURL); e == nil {
			logger.Infof(c, "[GetUploadToken] upload URL host=%q (browser PUT must use same Host)", u.Host)
		}
	}

	response.OkWithData(c, resp)
}

// BatchGetUploadToken 批量获取上传凭证
// @Summary 批量获取上传凭证
// @Description 批量获取多个文件的 presigned_url 上传凭证。如果某个文件未提供 router，将使用默认路由：/{username}/default
// @Tags 存储管理
// @Accept json
// @Produce json
// @Param request body dto.BatchGetUploadTokenReq true "批量获取上传凭证请求"
// @Success 200 {object} dto.BatchGetUploadTokenResp "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 500 {string} string "服务器内部错误"
// @Router /storage/api/v1/batch_upload_token [post]
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
		uploadSource := defaultUploadSource
		if fileReq.UploadSource != "" {
			uploadSource = getDefaultUploadSource(fileReq.UploadSource)
		}

		// 如果未提供 Router，使用默认路由：/{username}/default
		router := fileReq.Router
		if router == "" {
			router = fmt.Sprintf("%s/default", username)
			logger.Infof(c, "Router not provided for file %s, using default router: %s", fileReq.FileName, router)
		}

		// 生成上传凭证
		bucket := fileReq.Bucket
		if bucket == "" {
			bucket = s.storageService.GetBucketName()
		}
		var creds *storage.UploadCredentials
		var key string
		var expire time.Time
		var err error
		if fileReq.PreviewForKey != "" {
			creds, key, expire, err = s.storageService.GeneratePreviewUploadToken(ctx, bucket, fileReq.PreviewForKey, fileReq.FileName, fileReq.ContentType, fileReq.FileSize, uploadSource)
		} else {
			creds, key, expire, err = s.storageService.GenerateUploadToken(ctx, bucket, router, fileReq.FileName, fileReq.ContentType, fileReq.FileSize, uploadSource)
		}
		if err != nil {
			// 单个文件失败，记录错误但继续处理其他文件
			logger.Errorf(c, "Failed to generate upload credentials: file_name_len=%d, err=%v", len(fileReq.FileName), err)
			continue
		}

		// 获取 CDN 域名和存储引擎类型
		cdnDomain := s.storageService.GetCDNDomain()
		storageType := s.storageService.GetStorageType()

		// 构建预期的下载URL
		downloadURL, serverDownloadURL, _, err := s.storageService.GetFileURLsInBucket(ctx, bucket, key)
		if err != nil {
			logger.Errorf(c, "Failed to generate download URLs: key_len=%d, err=%v", len(key), err)
			// 下载URL生成失败不影响上传，设置为空
			downloadURL = ""
			serverDownloadURL = ""
		}

		// 构建响应
		token := buildUploadTokenResponse(creds, bucket, key, expire, cdnDomain, storageType, downloadURL, serverDownloadURL, username)
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
// @Router /storage/api/v1/upload_complete [post]
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
	var thumbnailURL string
	bucket := req.Bucket
	if bucket == "" {
		bucket = s.storageService.GetBucketName()
	}
	ref := s.storageService.BuildFileRef(bucket, req.Key)
	if req.Success {
		// 获取用户信息并创建上传记录
		requestUser := contextx.GetRequestUser(c)
		if err := createUploadRecord(
			s.storageService,
			ctx,
			bucket,
			req.Key,
			req.Router,
			req.FileName,
			req.Description,
			req.FileSize,
			req.ContentType,
			req.Hash,
			req.ThumbnailRef,
			req.PreviewKind,
			requestUser,
		); err != nil {
			logger.Errorf(c, "Failed to record upload to database: %v (file_key_len=%d)", err, len(req.Key))
			// 不影响响应，只记录错误（文件已上传到MinIO，只是数据库记录失败）
		}

		// 构建下载URL
		var expire time.Time
		var err error
		downloadURL, serverDownloadURL, expire, err = s.storageService.GetFileURLsInBucket(ctx, bucket, req.Key)
		if err != nil {
			logger.Errorf(c, "Failed to generate download URLs: %v", err)
			downloadURL = ""
			serverDownloadURL = ""
		}
		thumbnailURL = s.browserURLForRef(ctx, req.ThumbnailRef)

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
		Key:               req.Key,
		Bucket:            bucket,
		Ref:               ref,
		FileName:          req.FileName,
		Description:       req.Description,
		FileSize:          req.FileSize,
		ContentType:       req.ContentType,
		Hash:              req.Hash,
		ThumbnailRef:      req.ThumbnailRef,
		ThumbnailURL:      thumbnailURL,
		PreviewKind:       req.PreviewKind,
		Storage:           s.storageService.GetStorageType(),
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
// @Router /storage/api/v1/batch_upload_complete [post]
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
		bucket := item.Bucket
		if bucket == "" {
			bucket = s.storageService.GetBucketName()
		}
		ref := s.storageService.BuildFileRef(bucket, item.Key)
		if item.Success {
			// 只有上传成功时才创建记录
			if err := createUploadRecord(
				s.storageService,
				ctx,
				bucket,
				item.Key,
				item.Router,
				item.FileName,
				item.Description,
				item.FileSize,
				item.ContentType,
				item.Hash,
				item.ThumbnailRef,
				item.PreviewKind,
				requestUser,
			); err != nil {
				logger.Errorf(c, "Failed to record upload: key_len=%d, err=%v", len(item.Key), err)
				results = append(results, dto.BatchUploadCompleteResult{
					Key:    item.Key,
					Status: "failed",
					Error:  fmt.Sprintf("记录失败: %v", err),
				})
				continue
			}

			// 构建下载URL
			downloadURL, serverDownloadURL, _, err := s.storageService.GetFileURLsInBucket(ctx, bucket, item.Key)
			if err != nil {
				logger.Errorf(c, "Failed to generate download URLs: key_len=%d, err=%v", len(item.Key), err)
				downloadURL = ""
				serverDownloadURL = ""
			}
			thumbnailURL := s.browserURLForRef(ctx, item.ThumbnailRef)

			logger.Infof(c, "Upload complete: key=%s, success=true", item.Key)

			results = append(results, dto.BatchUploadCompleteResult{
				Key:               item.Key,
				Bucket:            bucket,
				Ref:               ref,
				Status:            "completed",
				DownloadURL:       downloadURL, // ✨ 外部访问的下载地址（前端使用）
				Description:       item.Description,
				ServerDownloadURL: serverDownloadURL, // ✨ 内部访问的下载地址（服务端使用）
				Hash:              item.Hash,         // ✨ 文件hash（用于 SDK 下载缓存）
				ThumbnailRef:      item.ThumbnailRef,
				ThumbnailURL:      thumbnailURL,
				PreviewKind:       item.PreviewKind,
			})
		} else {
			// 上传失败，不记录数据库，只记录日志
			logger.Warnf(c, "Upload failed: key=%s, error=%s", item.Key, item.Error)
			results = append(results, dto.BatchUploadCompleteResult{
				Key:    item.Key,
				Bucket: bucket,
				Ref:    ref,
				Status: "failed",
				Error:  item.Error,
			})
		}
	}

	response.OkWithData(c, dto.BatchUploadCompleteResp{
		Results: results,
	})
}

func (s *Storage) browserURLForRef(ctx context.Context, ref string) string {
	if ref == "" {
		return ""
	}
	bucket, key, err := s.storageService.ParseFileRef(ref)
	if err != nil {
		logger.Warnf(ctx, "Failed to parse file preview ref %q: %v", ref, err)
		return ""
	}
	browserURL, _, _, err := s.storageService.GetFileURLsInBucket(ctx, bucket, key)
	if err != nil {
		logger.Warnf(ctx, "Failed to generate file preview URL for ref %q: %v", ref, err)
		return ""
	}
	return browserURL
}

// ResolveFileRefs 批量解析文件引用，返回元数据和直连 URL。
// 前端使用 download_url 直连对象存储；SDK/容器使用 server_download_url 直连内部对象存储。
func (s *Storage) ResolveFileRefs(c *gin.Context) {
	var req dto.ResolveFileRefsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	files, err := s.storageService.ResolveFileRefs(ctx, req.Refs, req.Audience)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	response.OkWithData(c, dto.ResolveFileRefsResp{Files: files})
}

// UpdateFileDescription 更新文件描述
// @Summary 更新文件描述
// @Description 更新文件引用对应的描述元数据
// @Tags 存储管理
// @Accept json
// @Produce json
// @Param request body dto.UpdateFileDescriptionReq true "更新文件描述请求"
// @Success 200 {object} dto.UpdateFileDescriptionResp "更新成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 500 {string} string "服务器内部错误"
// @Router /storage/api/v1/files/description [post]
func (s *Storage) UpdateFileDescription(c *gin.Context) {
	var req dto.UpdateFileDescriptionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	resp, err := s.storageService.UpdateFileDescription(ctx, req.Ref, req.Bucket, req.Key, req.Description)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	response.OkWithData(c, resp)
}

// DownloadFile 获取文件（重定向到对象存储直连地址）
// @Summary 下载文件
// @Description 重定向到对象存储直连地址，避免服务端转发文件流
// @Tags 存储管理
// @Accept json
// @Produce application/octet-stream
// @Param key path string true "文件 Key"
// @Success 200 {file} file "文件内容"
// @Failure 400 {string} string "请求参数错误"
// @Failure 500 {string} string "服务器内部错误"
// @Router /storage/api/v1/download/{key} [get]
func (s *Storage) DownloadFile(c *gin.Context) {
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
	_, err := s.storageService.GetFileInfo(ctx, key)
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
	go func(parent context.Context) {
		writeCtx, cancel := context.WithTimeout(context.WithoutCancel(parent), 5*time.Second)
		defer cancel()
		if err := s.storageService.RecordDownload(writeCtx, downloadRecord); err != nil {
			logger.Errorf(writeCtx, "Failed to record download: %v", err)
			// 不影响下载流程，只记录错误
		}
	}(ctx)

	downloadURL, _, _, err := s.storageService.GetFileURLs(ctx, key)
	if err != nil || downloadURL == "" {
		logger.Errorf(c, "Failed to resolve download URL: %v", err)
		response.FailWithMessage(c, "生成下载链接失败")
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, downloadURL)
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
// @Router /storage/api/v1/files/{key} [delete]
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
// @Router /storage/api/v1/files/{key}/info [get]
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
// @Param router query string true "函数路径，例如：luobei/test88888/plugins/cashier_desk"
// @Success 200 {object} dto.GetStorageStatsResp "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 500 {string} string "服务器内部错误"
// @Router /storage/api/v1/stats [get]
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
// @Router /storage/api/v1/files [get]
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
// @Router /storage/api/v1/batch_delete [post]
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
