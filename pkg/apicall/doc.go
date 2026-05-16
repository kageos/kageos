package apicall

import (
	"context"

	"github.com/ai-agent-os/ai-agent-os/dto"
)

// GetDocsByPaths 根据路径批量获取文档（agent-server -> app-server）
// paths: 文档路径列表，直接根据 full_code_path IN 查询
// 例如：["/user/myapp/docs"]
func GetDocsByPaths(ctx context.Context, paths []string) (*dto.GetDocsByPathsResp, error) {
	if len(paths) == 0 {
		return &dto.GetDocsByPathsResp{Docs: []*dto.DocItem{}}, nil
	}

	resp, err := GetAPI[*dto.BatchGetDocsResp](ctx, "/workspace/api/v1/docs/batch", buildQueryParams(
		withRepeatedQueryValues("paths", paths),
		withBoolLiteralQueryValue("include_content", true),
	))
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
	resp, err := GetAPI[*dto.BatchGetDocsResp](ctx, "/workspace/api/v1/docs/batch", buildQueryParams(
		withRepeatedQueryValues("paths", paths),
		withBoolLiteralQueryValue("include_content", false),
	))
	if err != nil {
		return nil, err
	}
	return &dto.GetDocsByPathsResp{Docs: resp.Docs}, nil
}

// GetDoc 根据完整路径获取单篇文档内容（用于 read_doc 工具按需拉取）
// fullCodePath: 文档完整路径，如 /user/myapp/docs/guide
func GetDoc(ctx context.Context, fullCodePath string) (*dto.DocItem, error) {
	if normalizeWorkspaceFunctionPath(fullCodePath) == "" {
		return nil, nil
	}
	path := buildWorkspaceInfoPath("/workspace/api/v1/docs/info", fullCodePath)
	return GetAPI[*dto.DocItem](ctx, path, nil)
}
