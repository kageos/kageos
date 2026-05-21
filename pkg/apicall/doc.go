package apicall

import (
	"context"

	"github.com/kageos/kageos/dto"
)

// GetDoc 根据完整路径获取单篇文档内容（用于 read_doc 工具按需拉取）
// fullCodePath: 文档完整路径，如 /user/myapp/docs/guide
func GetDoc(ctx context.Context, fullCodePath string) (*dto.DocItem, error) {
	if normalizeWorkspaceFunctionPath(fullCodePath) == "" {
		return nil, nil
	}
	path := buildWorkspaceInfoPath("/workspace/api/v1/docs/info", fullCodePath)
	return GetAPI[*dto.DocItem](ctx, path, nil)
}
