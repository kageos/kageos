package v1

import (
	"fmt"
	"strings"
	"time"

	"github.com/kageos/kageos/pkg/contextx"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/core/app-storage/storage"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/ginx/response"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/pkg/publicshare"
)

func (s *Storage) publicShareRequestUser(c *gin.Context, router string) (string, bool) {
	claims, err := publicshare.ValidateAnonymousToken(c.GetHeader(publicshare.AnonymousTokenHeader))
	if err != nil {
		response.NoAuth(c, "匿名访问凭证无效，请刷新页面后重试")
		return "", false
	}
	shareID := strings.TrimSpace(c.Param("share_id"))
	if shareID == "" {
		response.NoAuth(c, "分享链接无效")
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
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}
	router := strings.Trim(strings.TrimSpace(req.Router), "/")
	if router == "" {
		response.BadRequest(c, "公开上传必须提供函数路由")
		return
	}
	requestUser, ok := s.publicShareRequestUser(c, router)
	if !ok {
		return
	}

	uploadSource := storage.UploadSourceBrowser
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
			response.Forbidden(c, "预览文件路径不属于当前分享")
			return
		}
		creds, key, expire, err = s.storageService.GeneratePreviewUploadToken(ctx, bucket, req.PreviewForKey, req.FileName, req.ContentType, req.FileSize, uploadSource)
	} else {
		creds, key, expire, err = s.storageService.GenerateUploadToken(ctx, bucket, router, req.FileName, req.ContentType, req.FileSize, uploadSource)
	}
	if err != nil {
		response.Error(c, err)
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
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}
	router := strings.Trim(strings.TrimSpace(req.Router), "/")
	if router == "" {
		response.BadRequest(c, "公开上传完成必须提供函数路由")
		return
	}
	if _, ok := s.publicShareRequestUser(c, router); !ok {
		return
	}
	if req.Success && !isObjectKeyInRouter(req.Key, router) {
		response.Forbidden(c, "文件路径不属于当前分享")
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
		response.BadRequest(c, "请求参数错误: "+err.Error())
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
		response.BadRequest(c, "公开上传完成必须提供函数路由")
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
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}
	if _, ok := s.publicShareRequestUser(c, ""); !ok {
		return
	}

	ctx := contextx.ToContext(c)
	files, err := s.storageService.ResolveFileRefs(ctx, req.Refs, req.Audience)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OkWithData(c, dto.ResolveFileRefsResp{Files: files})
}
