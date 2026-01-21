package apicall

import (
	"context"

	"github.com/ai-agent-os/ai-agent-os/dto"
)

// GetDocsByPaths 根据路径批量获取文档（agent-server -> app-server）
// paths: 文档路径列表，直接根据 full_code_path IN 查询
// 例如：["/system/official/sdk", "/user/myapp/docs"]
func GetDocsByPaths(ctx context.Context, paths []string) (*dto.GetDocsByPathsResp, error) {
	if len(paths) == 0 {
		return &dto.GetDocsByPathsResp{Docs: []*dto.DocItem{}}, nil
	}
	
	// 构建请求体
	req := dto.GetDocsByPathsReq{
		Paths: paths,
	}
	
	// POST 请求（避免 Gin wildcard 路由冲突）
	return PostAPI[dto.GetDocsByPathsReq, *dto.GetDocsByPathsResp](ctx, "/workspace/api/v1/docs/query", req)
}
