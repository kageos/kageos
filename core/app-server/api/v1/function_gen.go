package v1

import (
	"github.com/ai-agent-os/ai-agent-os/core/app-server/service"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/response"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/gin-gonic/gin"
)

// FunctionGen 函数生成相关 API
type FunctionGen struct {
	appService *service.AppService
	server     FunctionGenServer // 需要访问 server 的方法
}

// FunctionGenServer 定义需要从 Server 访问的方法接口
type FunctionGenServer interface {
	HandleFunctionGenResult(ctx *gin.Context, req *dto.AddFunctionsReq)
}

// NewFunctionGen 创建 FunctionGen 处理器
func NewFunctionGen(appService *service.AppService, server FunctionGenServer) *FunctionGen {
	return &FunctionGen{
		appService: appService,
		server:     server,
	}
}

// ReceiveFunctionGenResult 接收函数生成结果（HTTP 接口，替代 NATS 订阅）
// @Summary 接收函数生成结果
// @Description 接收来自 agent-server 的函数生成结果，处理并创建函数
// @Tags 函数生成
// @Accept json
// @Produce json
// @Param X-Trace-Id header string false "追踪ID"
// @Param X-Request-User header string false "请求用户"
// @Param request body dto.AddFunctionsReq true "添加函数请求"
// @Success 200 {object} map[string]interface{} "处理成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/function_gen/result [post]
func (f *FunctionGen) ReceiveFunctionGenResult(c *gin.Context) {
	var req dto.AddFunctionsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Errorf(c, "[FunctionGen API] 解析请求失败: %v", err)
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	// 调用 server 的处理方法
	f.server.HandleFunctionGenResult(c, &req)
	response.OkWithData(c, map[string]interface{}{
		"message": "函数生成结果已接收并处理",
	})
}
