package v1

import (
	"context"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/response"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/gin-gonic/gin"
)

// FunctionGen 函数生成相关 API
type FunctionGen struct {
	functionGenService FunctionGenService // 直接使用 service 层
}

// FunctionGenService 定义需要从 Service 访问的方法接口
type FunctionGenService interface {
	ProcessFunctionGenCallback(ctx context.Context, callback *dto.FunctionGenCallback) error
}

// NewFunctionGen 创建 FunctionGen 处理器
func NewFunctionGen(functionGenService FunctionGenService) *FunctionGen {
	return &FunctionGen{
		functionGenService: functionGenService,
	}
}

// ReceiveCallback 接收工作空间更新回调（HTTP 接口，替代 NATS 订阅）
// @Summary 接收工作空间更新回调
// @Description 接收来自 app-server 的工作空间更新回调，更新生成记录状态
// @Tags 工作空间
// @Accept json
// @Produce json
// @Param X-Trace-Id header string false "追踪ID"
// @Param X-Request-User header string false "请求用户"
// @Param request body dto.FunctionGenCallback true "工作空间更新回调"
// @Success 200 {object} map[string]interface{} "处理成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 500 {string} string "服务器内部错误"
// @Router /agent/api/v1/workspace/update/callback [post]
func (f *FunctionGen) ReceiveCallback(c *gin.Context) {
	var callback dto.FunctionGenCallback
	if err := c.ShouldBindJSON(&callback); err != nil {
		logger.Errorf(c, "[FunctionGen API] 解析请求失败: %v", err)
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	// 直接调用 service 层处理
	ctx := contextx.ToContext(c)
	if err := f.functionGenService.ProcessFunctionGenCallback(ctx, &callback); err != nil {
		logger.Errorf(c, "[FunctionGen API] 处理回调失败: %v", err)
		response.FailWithMessage(c, "处理回调失败: "+err.Error())
		return
	}

	response.OkWithData(c, map[string]interface{}{
		"message": "回调已接收并处理",
	})
}

