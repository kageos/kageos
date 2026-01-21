package apicall

import (
	"context"
	"net/url"
	"strings"
)

// GetDocsByPathsReq 根据路径批量获取文档请求
type GetDocsByPathsReq struct {
	Paths []string `json:"paths"` // 文档路径列表，如 ["/system/official/sdk", "/user/myapp/docs"]
}

// DocItem 文档项
type DocItem struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Content      string `json:"content"`
	Format       string `json:"format"`
	FullCodePath string `json:"full_code_path"`
	Summary      string `json:"summary"`
	Category     string `json:"category"`
}

// GetDocsByPathsResp 根据路径批量获取文档响应
type GetDocsByPathsResp struct {
	Docs []*DocItem `json:"docs"` // 文档列表
}

// GetDocsByPaths 根据路径批量获取文档（agent-server -> app-server）
// paths: 文档路径列表，支持路径前缀匹配
// 例如：["/system/official/sdk", "/user/myapp/docs"]
// 会返回这些路径及其子路径下的所有文档
func GetDocsByPaths(ctx context.Context, paths []string) (*GetDocsByPathsResp, error) {
	if len(paths) == 0 {
		return &GetDocsByPathsResp{Docs: []*DocItem{}}, nil
	}
	
	// 构建查询参数
	params := url.Values{}
	params.Set("paths", strings.Join(paths, ","))
	
	return GetAPI[*GetDocsByPathsResp](ctx, "/workspace/api/v1/docs/batch", params)
}
