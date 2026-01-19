package apicall

import (
	"context"

	"github.com/ai-agent-os/ai-agent-os/dto"
)

// NotifyWorkspaceUpdateComplete 通知工作空间更新完成（workspace -> agent-server）
// workspace 更新完毕后，将更新的信息回调给 agent-server
func NotifyWorkspaceUpdateComplete(ctx context.Context, req *dto.FunctionGenCallback) error {
	_, err := PostAPI[*dto.FunctionGenCallback, *map[string]interface{}](ctx, "/agent/api/v1/workspace/update/callback", req)
	return err
}
