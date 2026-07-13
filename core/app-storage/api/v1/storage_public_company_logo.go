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
)

func (s *Storage) GetPublicCompanyLogoUploadToken(c *gin.Context) {
	var req dto.GetUploadTokenReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}
	if err := validatePublicCompanyLogo(req.FileName, req.ContentType, req.FileSize); err != nil {
		response.Error(c, err)
		return
	}

	ctx := contextx.ToContext(c)
	bucket := req.Bucket
	if bucket == "" {
		bucket = s.storageService.GetBucketName()
	}
	creds, key, expire, err := s.storageService.GenerateUploadToken(
		ctx,
		bucket,
		publicCompanyLogoRouter,
		req.FileName,
		req.ContentType,
		req.FileSize,
		storage.UploadSourceBrowser,
	)
	if err != nil {
		response.Error(c, err)
		return
	}

	downloadURL, serverDownloadURL, _, err := s.storageService.GetFileURLsInBucket(ctx, bucket, key)
	if err != nil {
		logger.Warnf(c, "Failed to generate public company logo download URLs: %v", err)
		downloadURL = ""
		serverDownloadURL = ""
	}

	resp := buildUploadTokenResponse(
		creds,
		bucket,
		key,
		expire,
		s.storageService.GetCDNDomain(),
		s.storageService.GetStorageType(),
		downloadURL,
		serverDownloadURL,
		publicCompanyLogoUser,
	)
	response.OkWithData(c, resp)
}

func (s *Storage) PublicCompanyLogoUploadComplete(c *gin.Context) {
	var req dto.UploadCompleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}
	if !strings.HasPrefix(strings.TrimLeft(req.Key, "/"), publicCompanyLogoRouter+"/") {
		response.BadRequest(c, "企业 Logo 文件路径不合法")
		return
	}
	if err := validatePublicCompanyLogo(req.FileName, req.ContentType, req.FileSize); err != nil {
		response.Error(c, err)
		return
	}

	ctx := contextx.ToContext(c)
	bucket := req.Bucket
	if bucket == "" {
		bucket = s.storageService.GetBucketName()
	}
	ref := s.storageService.BuildFileRef(bucket, req.Key)
	var downloadURL string
	var serverDownloadURL string
	var expireStr string
	if req.Success {
		if err := createUploadRecord(
			s.storageService,
			ctx,
			bucket,
			req.Key,
			publicCompanyLogoRouter,
			req.FileName,
			"company logo",
			req.FileSize,
			req.ContentType,
			req.Hash,
			"",
			"",
			publicCompanyLogoUser,
		); err != nil {
			logger.Warnf(c, "Failed to record public company logo upload: %v", err)
		}
		expire := time.Time{}
		var err error
		downloadURL, serverDownloadURL, expire, err = s.storageService.GetFileURLsInBucket(ctx, bucket, req.Key)
		if err != nil {
			logger.Errorf(c, "Failed to generate public company logo URL: %v", err)
			downloadURL = ""
			serverDownloadURL = ""
		}
		if !expire.IsZero() {
			expireStr = expire.Format(storage.TimeFormat)
		}
	}

	response.OkWithData(c, &dto.UploadCompleteResp{
		Message:           "企业 Logo 上传完成",
		Key:               req.Key,
		Bucket:            bucket,
		Ref:               ref,
		FileName:          req.FileName,
		Description:       "company logo",
		FileSize:          req.FileSize,
		ContentType:       req.ContentType,
		Hash:              req.Hash,
		Storage:           s.storageService.GetStorageType(),
		DownloadURL:       downloadURL,
		ServerDownloadURL: serverDownloadURL,
		Expire:            expireStr,
	})
}

func validatePublicCompanyLogo(fileName string, contentType string, fileSize int64) error {
	if strings.TrimSpace(fileName) == "" {
		return fmt.Errorf("文件名不能为空")
	}
	if fileSize <= 0 {
		return fmt.Errorf("文件大小不能为空")
	}
	if fileSize > publicCompanyLogoMaxSize {
		return fmt.Errorf("企业 Logo 不能超过 512KB")
	}
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	allowed := map[string]bool{
		"image/png":     true,
		"image/jpeg":    true,
		"image/jpg":     true,
		"image/webp":    true,
		"image/svg+xml": true,
	}
	if !allowed[contentType] {
		return fmt.Errorf("企业 Logo 仅支持 PNG、JPG、WebP 或 SVG")
	}
	return nil
}
