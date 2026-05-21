package apicall

import (
	"context"
	"net/url"

	"github.com/kageos/kageos/dto"
)

// BatchGetUploadToken 批量获取上传凭证
func BatchGetUploadToken(ctx context.Context, req *dto.BatchGetUploadTokenReq) (*dto.BatchGetUploadTokenResp, error) {
	return PostAPI[*dto.BatchGetUploadTokenReq, *dto.BatchGetUploadTokenResp](ctx, "/storage/api/v1/batch_upload_token", req)
}

// BatchUploadComplete 批量通知上传完成
func BatchUploadComplete(ctx context.Context, req *dto.BatchUploadCompleteReq) (*dto.BatchUploadCompleteResp, error) {
	return PostAPI[*dto.BatchUploadCompleteReq, *dto.BatchUploadCompleteResp](ctx, "/storage/api/v1/batch_upload_complete", req)
}

func ResolveFileRefs(ctx context.Context, req *dto.ResolveFileRefsReq) (*dto.ResolveFileRefsResp, error) {
	return PostAPI[*dto.ResolveFileRefsReq, *dto.ResolveFileRefsResp](ctx, "/storage/api/v1/files/resolve", req)
}

func ResolvePublicShareFileRefs(ctx context.Context, shareID string, req *dto.ResolveFileRefsReq) (*dto.ResolveFileRefsResp, error) {
	path := "/storage/api/v1/public/share/" + url.PathEscape(shareID) + "/files/resolve"
	return PostAPI[*dto.ResolveFileRefsReq, *dto.ResolveFileRefsResp](ctx, path, req)
}
