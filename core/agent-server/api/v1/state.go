package v1

import (
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/service"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/response"
	"github.com/gin-gonic/gin"
)

type State struct {
	runtimeState service.RuntimeStateStore
}

func NewState(runtimeState service.RuntimeStateStore) *State {
	return &State{runtimeState: runtimeState}
}

// RuntimeSummary 返回按 full_code_path 聚合后的运行态摘要。
// GET /agent/api/v1/state/runtime-summary?root_full_code_path=/user/app
func (h *State) RuntimeSummary(c *gin.Context) {
	if h.runtimeState == nil {
		response.OkWithData(c, dto.RuntimeStateSummaryResp{Summaries: map[string]dto.RuntimeStateSummary{}})
		return
	}
	ctx := contextx.ToContext(c)
	summaries, err := h.runtimeState.Summary(ctx, service.RuntimeStateFilter{
		RootFullCodePath: c.Query("root_full_code_path"),
		Kind:             c.Query("kind"),
		Status:           c.Query("status"),
	})
	if err != nil {
		response.FailWithMessage(c, "获取运行态摘要失败: "+err.Error())
		return
	}
	response.OkWithData(c, dto.RuntimeStateSummaryResp{Summaries: summaries})
}

// RuntimeItems 返回当前运行态明细，用于点击服务树 badge 后展示会话列表。
// GET /agent/api/v1/state/runtime-items?root_full_code_path=/user/app
func (h *State) RuntimeItems(c *gin.Context) {
	if h.runtimeState == nil {
		response.OkWithData(c, dto.RuntimeStateItemsResp{Items: []dto.RuntimeStateItem{}})
		return
	}
	ctx := contextx.ToContext(c)
	items, err := h.runtimeState.List(ctx, service.RuntimeStateFilter{
		RootFullCodePath: c.Query("root_full_code_path"),
		Kind:             c.Query("kind"),
		Status:           c.Query("status"),
	})
	if err != nil {
		response.FailWithMessage(c, "获取运行态明细失败: "+err.Error())
		return
	}
	response.OkWithData(c, dto.RuntimeStateItemsResp{Items: items})
}
