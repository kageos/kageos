package apicall

import (
	"context"
	"net/url"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
)

// GetDocsByPaths 根据路径批量获取文档（agent-server -> app-server）
// paths: 文档路径列表，直接根据 full_code_path IN 查询
// 例如：["/system/official/sdk", "/user/myapp/docs"]
func GetDocsByPaths(ctx context.Context, paths []string) (*dto.GetDocsByPathsResp, error) {
	if len(paths) == 0 {
		return &dto.GetDocsByPathsResp{Docs: []*dto.DocItem{}}, nil
	}

	// 构建查询参数（使用批量查询接口）
	queryParams := url.Values{}
	for _, path := range paths {
		queryParams.Add("paths", path) // 使用 Add 支持多个 paths 参数
	}
	queryParams.Set("include_content", "true") // 智能体需要完整内容

	// GET 请求，使用 query 参数
	resp, err := GetAPI[*dto.BatchGetDocsResp](ctx, "/workspace/api/v1/docs/batch", queryParams)
	if err != nil {
		return nil, err
	}

	// 转换为旧格式（向后兼容）
	return &dto.GetDocsByPathsResp{Docs: resp.Docs}, nil
}

// GetDocsCatalogByPaths 根据路径批量获取文档目录（仅元数据，不含 content，用于按需加载说明）
// 返回的 DocItem 仅含 Name、FullCodePath、Summary 等，Content 为空
func GetDocsCatalogByPaths(ctx context.Context, paths []string) (*dto.GetDocsByPathsResp, error) {
	if len(paths) == 0 {
		return &dto.GetDocsByPathsResp{Docs: []*dto.DocItem{}}, nil
	}
	queryParams := url.Values{}
	for _, path := range paths {
		queryParams.Add("paths", path)
	}
	queryParams.Set("include_content", "false")
	resp, err := GetAPI[*dto.BatchGetDocsResp](ctx, "/workspace/api/v1/docs/batch", queryParams)
	if err != nil {
		return nil, err
	}
	return &dto.GetDocsByPathsResp{Docs: resp.Docs}, nil
}

// GetDoc 根据完整路径获取单篇文档内容（用于 read_doc 工具按需拉取）
// fullCodePath: 文档完整路径，如 /system/official/sdk
func GetDoc(ctx context.Context, fullCodePath string) (*dto.DocItem, error) {
	fullCodePath = strings.TrimSpace(fullCodePath)
	if fullCodePath == "" {
		return nil, nil
	}
	// 路由为 GET /info/*full-code-path，路径段不含前导斜杠，如 system/official/sdk
	pathSeg := strings.Trim(fullCodePath, "/")
	path := "/workspace/api/v1/docs/info/" + pathSeg
	return GetAPI[*dto.DocItem](ctx, path, nil)
}
