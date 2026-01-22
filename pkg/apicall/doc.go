package apicall

import (
	"context"
	"net/url"
	"strconv"

	"github.com/ai-agent-os/ai-agent-os/dto"
)

// GetDocsByPaths 根据路径批量获取文档（agent-server -> app-server）
// paths: 文档路径列表，直接根据 full_code_path IN 查询
// 例如：["/system/official/sdk", "/user/myapp/docs"]
func GetDocsByPaths(ctx context.Context, paths []string) (*dto.GetDocsByPathsResp, error) {
	if len(paths) == 0 {
		return &dto.GetDocsByPathsResp{Docs: []*dto.DocItem{}}, nil
	}
	
	// 构建查询参数（使用新的统一接口）
	queryParams := url.Values{}
	for _, path := range paths {
		queryParams.Add("paths", path) // 使用 Add 支持多个 paths 参数
	}
	queryParams.Set("include_content", "true") // 智能体需要完整内容
	
	// GET 请求，使用 query 参数
	resp, err := GetAPI[*dto.QueryDocsResp](ctx, "/workspace/api/v1/docs/query", queryParams)
	if err != nil {
		return nil, err
	}
	
	// 转换为旧格式（向后兼容）
	return &dto.GetDocsByPathsResp{Docs: resp.Docs}, nil
}
