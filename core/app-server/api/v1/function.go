package v1

import (
	"strings"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/service"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/response"
	"github.com/gin-gonic/gin"
)

type Function struct {
	functionService *service.FunctionService
}

// NewFunction 创建 Function 处理器（依赖注入）
func NewFunction(functionService *service.FunctionService) *Function {
	return &Function{
		functionService: functionService,
	}
}

// GetFunction 获取函数详情
// @Summary 获取函数详情
// @Description 根据函数类型和 full-code-path 获取函数的详细信息
// @Tags 函数管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param func-type path string true "函数类型：table、form、chart"
// @Param full-code-path path string true "函数完整路径，如 /luobei/operations/crm/ticket"
// @Success 200 {object} dto.GetFunctionResp "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 403 {string} string "权限不足"
// @Failure 404 {string} string "函数不存在"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/function/info/{func-type}/{full-code-path} [get]
func (f *Function) GetFunction(c *gin.Context) {
	//// ⭐ 从路径参数获取函数类型和 full-code-path
	//funcType := c.Param("func-type")
	fullCodePath := c.Param("full-code-path")

	if fullCodePath == "" {
		response.FailWithMessage(c, "缺少full-code-path参数")
		return
	}

	// 确保路径以 / 开头
	if !strings.HasPrefix(fullCodePath, "/") {
		fullCodePath = "/" + fullCodePath
	}

	ctx := contextx.ToContext(c)

	// 获取函数详情
	resp, err := f.functionService.GetFunctionByFullCodePath(ctx, fullCodePath)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	response.OkWithData(c, resp)
}
