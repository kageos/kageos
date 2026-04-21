package apicall

import (
	"context"

	"github.com/ai-agent-os/ai-agent-os/dto"
)

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

func ResolveFileRefs(ctx context.Context, req *dto.ResolveFileRefsReq) (*dto.ResolveFileRefsResp, error) {
	return PostAPI[*dto.ResolveFileRefsReq, *dto.ResolveFileRefsResp](ctx, "/storage/api/v1/files/resolve", req)
}
