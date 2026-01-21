package v1

import (
	"github.com/ai-agent-os/ai-agent-os/core/app-server/service"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/response"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/gin-gonic/gin"
)

// FunctionGen 函数生成相关 API（已废弃，保留用于兼容）
type FunctionGen struct {
	serviceTreeService *service.ServiceTreeService
}

// NewFunctionGen 创建 FunctionGen 处理器
func NewFunctionGen(serviceTreeService *service.ServiceTreeService) *FunctionGen {
	return &FunctionGen{
		serviceTreeService: serviceTreeService,
	}
}

// ReceiveFunctionGenResult 已废弃，请使用 ServiceTree.AddFunctions
// 该接口已迁移到 /workspace/api/v1/service_tree/add_functions
// @Deprecated 使用 ServiceTree.AddFunctions 替代
func (f *FunctionGen) ReceiveFunctionGenResult(c *gin.Context) {
	var req dto.AddFunctionsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Errorf(c, "[FunctionGen API] 解析请求失败: %v", err)
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}

	// 直接调用 ServiceTreeService.ProcessFunctionGenResult（兼容旧接口）
	ctx := c.Request.Context()
	if err := f.serviceTreeService.ProcessFunctionGenResult(ctx, &req); err != nil {
		logger.Errorf(c, "[FunctionGen API] 处理失败: %v", err)
		response.FailWithMessage(c, "处理失败: "+err.Error())
		return
	}

	response.OkWithData(c, map[string]interface{}{
		"message": "函数生成结果已接收并处理",
	})
}
