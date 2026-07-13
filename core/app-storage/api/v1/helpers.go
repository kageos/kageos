package v1

import (
	"context"
	"strings"
	"time"

	"github.com/kageos/kageos/core/app-storage/model"
	"github.com/kageos/kageos/core/app-storage/service"
	"github.com/kageos/kageos/core/app-storage/storage"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/logger"
)

func logUploadTokenDebug(ctx context.Context, operation string, req dto.GetUploadTokenReq, resp *dto.GetUploadTokenResp, err error) {
	method := ""
	issued := false
	uploadURLIssued := false
	serverUploadURLIssued := false
	downloadURLIssued := false
	serverDownloadURLIssued := false
	if resp != nil {
		method = string(resp.Method)
		issued = true
		uploadURLIssued = resp.UploadURL != ""
		serverUploadURLIssued = resp.ServerUploadURL != ""
		downloadURLIssued = resp.DownloadURL != ""
		serverDownloadURLIssued = resp.ServerDownloadURL != ""
	}
	logger.Debugf(
		ctx,
		"%s file_name_len=%d content_type=%s file_size=%d router_set=%v bucket_set=%v preview_for_key_set=%v issued=%v method=%s upload_url_issued=%v server_upload_url_issued=%v download_url_issued=%v server_download_url_issued=%v err:%v",
		operation,
		len(req.FileName),
		req.ContentType,
		req.FileSize,
		req.Router != "",
		req.Bucket != "",
		req.PreviewForKey != "",
		issued,
		method,
		uploadURLIssued,
		serverUploadURLIssued,
		downloadURLIssued,
		serverDownloadURLIssued,
		err,
	)
}

// buildUploadTokenResponse 构建上传凭证响应
func buildUploadTokenResponse(
	creds *storage.UploadCredentials,
	bucket string,
	key string,
	expire time.Time,
	cdnDomain string,
	storageType string,
	downloadURL string,
	serverDownloadURL string,
	username string,
) *dto.GetUploadTokenResp {
	ref := ""
	if bucket != "" && key != "" {
		ref = strings.Trim(bucket, "/") + "/" + strings.TrimLeft(key, "/")
	}
	return &dto.GetUploadTokenResp{
		Key:               key,
		Bucket:            bucket,
		Ref:               ref,
		Expire:            expire.Format(storage.TimeFormat),
		Method:            dto.UploadMethod(creds.Method),
		Storage:           storageType,
		UploadURL:         creds.UploadURL,
		ServerUploadURL:   creds.ServerUploadURL,
		Headers:           creds.Headers,
		UploadHost:        creds.UploadHost,
		UploadDomain:      creds.UploadDomain,
		CDNDomain:         cdnDomain,
		DownloadURL:       downloadURL,
		ServerDownloadURL: serverDownloadURL,
		Username:          username,
	}
}

// createUploadRecord 创建上传记录
func createUploadRecord(
	storageService *service.StorageService,
	ctx context.Context,
	bucket string,
	key string,
	router string,
	fileName string,
	description string,
	fileSize int64,
	contentType string,
	hash string,
	thumbnailRef string,
	previewKind string,
	username string,
) error {
	tenant := extractTenantFromRouter(router)
	uploadRecord := &model.FileUpload{
		Bucket:       bucket,
		FileKey:      key,
		Router:       router,
		FileName:     fileName,
		Description:  description,
		FileSize:     fileSize,
		ContentType:  contentType,
		Hash:         hash,
		ThumbnailRef: thumbnailRef,
		PreviewKind:  previewKind,
		Username:     username,
		Tenant:       tenant,
		Status:       "completed",
	}
	return storageService.RecordUpload(ctx, uploadRecord)
}

// extractTenantFromRouter 从 router 中提取 tenant（第一个路径段）
func extractTenantFromRouter(router string) string {
	parts := strings.Split(router, "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}
