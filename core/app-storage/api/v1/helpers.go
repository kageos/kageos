package v1

import (
	"context"
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/app-storage/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-storage/service"
	"github.com/ai-agent-os/ai-agent-os/core/app-storage/storage"
	"github.com/ai-agent-os/ai-agent-os/dto"
)

// getDefaultUploadSource 获取默认上传来源，如果为空则返回browser
func getDefaultUploadSource(uploadSource dto.UploadSource) string {
	if uploadSource == "" {
		return string(dto.UploadSourceBrowser)
	}
	return string(uploadSource)
}

// buildUploadTokenResponse 构建上传凭证响应
func buildUploadTokenResponse(
	creds *storage.UploadCredentials,
	key string,
	expire time.Time,
	cdnDomain string,
	storageType string,
	downloadURL string,
	serverDownloadURL string,
	username string,
) *dto.GetUploadTokenResp {
	return &dto.GetUploadTokenResp{
		Key:               key,
		Bucket:            "", // 会在调用处设置
		Expire:            expire.Format(storage.TimeFormat),
		Method:            dto.UploadMethod(creds.Method),
		Storage:           storageType,
		URL:               creds.URL,
		ServerURL:         creds.ServerURL,
		Headers:           creds.Headers,
		UploadHost:        creds.UploadHost,
		UploadDomain:      creds.UploadDomain,
		FormData:          creds.FormData,
		PostURL:           creds.PostURL,
		SDKConfig:         creds.SDKConfig,
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
	key string,
	router string,
	fileName string,
	fileSize int64,
	contentType string,
	hash string,
	username string,
) error {
	tenant := extractTenantFromRouter(router)
	uploadRecord := &model.FileUpload{
		FileKey:     key,
		Router:      router,
		FileName:    fileName,
		FileSize:    fileSize,
		ContentType: contentType,
		Hash:        hash,
		UserID:      nil,
		Username:    username,
		Tenant:      tenant,
		Status:      "completed",
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

