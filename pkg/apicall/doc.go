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
	
	// 构建请求体（使用新的统一接口）
	req := dto.QueryDocsReq{
		Paths:          paths,
		IncludeContent: true, // 智能体需要完整内容
	}
	
	// POST 请求
	resp, err := PostAPI[dto.QueryDocsReq, *dto.QueryDocsResp](ctx, "/workspace/api/v1/docs/query", req)
	if err != nil {
		return nil, err
	}
	
	// 转换为旧格式（向后兼容）
	return &dto.GetDocsByPathsResp{Docs: resp.Docs}, nil
}
