package v1

import (
	"fmt"
	"net/url"
	"time"

	"github.com/kageos/kageos/pkg/contextx"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/core/app-storage/storage"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/ginx/response"
	"github.com/kageos/kageos/pkg/logger"
)

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
// @Router /storage/api/v1/upload-token [post]
func (s *Storage) GetUploadToken(c *gin.Context) {
	var req dto.GetUploadTokenReq
	var resp *dto.GetUploadTokenResp
	var err error
	defer func() {
		logUploadTokenDebug(c, "GetUploadToken", req, resp, err)
	}()

	// 绑定请求参数
	if err = c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	// 获取当前登录用户的用户名
	username := contextx.GetRequestUser(c)

	// 如果未提供 Router，使用默认路由：/{username}/default
	router := req.Router
	if router == "" {
		if username == "" {
			response.NoAuth(c, "未提供路由且无法获取用户信息")
			return
		}
		router = fmt.Sprintf("%s/default", username)
		logger.Infof(c, "Router not provided, using default router: %s", router)
	}

	// 设置默认上传来源
	uploadSource := storage.UploadSourceBrowser

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
		response.Error(c, err)
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
// @Router /storage/api/v1/batch-upload-token [post]
func (s *Storage) BatchGetUploadToken(c *gin.Context) {
	var req dto.BatchGetUploadTokenReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	// 获取当前登录用户的用户名
	username := contextx.GetRequestUser(c)
	if username == "" {
		response.NoAuth(c, "无法获取用户信息")
		return
	}

	// 将 gin.Context 转换为标准 context.Context
	ctx := contextx.ToContext(c)

	// 批量生成上传凭证
	tokens := make([]dto.GetUploadTokenResp, 0, len(req.Files))
	for _, fileReq := range req.Files {
		uploadSource := storage.UploadSourceBrowser

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
// @Router /storage/api/v1/upload-complete [post]
func (s *Storage) UploadComplete(c *gin.Context) {
	var req dto.UploadCompleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误: "+err.Error())
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
// @Router /storage/api/v1/batch-upload-complete [post]
func (s *Storage) BatchUploadComplete(c *gin.Context) {
	var req dto.BatchUploadCompleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误: "+err.Error())
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
