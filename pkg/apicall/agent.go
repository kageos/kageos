package apicall

import (
	"net/http"

	"github.com/ai-agent-os/ai-agent-os/dto"
)

// NotifyWorkspaceUpdateComplete 通知工作空间更新完成（workspace -> agent-server）
// workspace 更新完毕后，将更新的信息回调给 agent-server
func NotifyWorkspaceUpdateComplete(header *Header, req *dto.FunctionGenCallback) error {
	_, err := callAPI[map[string]interface{}](
		http.MethodPost,
		"/agent/api/v1/workspace/update/callback",
		header,
		req,
	)
	return err
}
