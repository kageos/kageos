package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/access"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/ginx/response"
	"github.com/kageos/kageos/pkg/logger"
)

// AddFunctions 向服务目录添加函数（服务间调用，不需要JWT验证）
// @Summary 向服务目录添加函数
// @Description 接收来自 agent-server 的 Go 源码，写入到工作空间对应目录；默认同步构建，skip_build=true 时仅写文件。
// @Tags 服务目录
// @Accept json
// @Produce json
// @Param X-Trace-Id header string false "追踪ID（用于链路追踪）"
// @Param X-Request-User header string false "请求用户（用于审计）"
// @Param X-Token header string false "Token（服务间调用时透传）"
// @Param request body dto.AddFunctionsReq true "添加函数请求"
// @Success 200 {object} dto.AddFunctionsResp "处理成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/functions/batch [post]
func (s *ServiceTree) AddFunctions(c *gin.Context) {
	var req dto.AddFunctionsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Errorf(c, "[ServiceTree API] 解析请求失败: %v", err)
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	if err := requireAccess(c, s.teamAccessService, req.FullCodePath, access.ActionAdmin); err != nil {
		response.Error(c, err)
		return
	}

	resp, err := s.serviceTreeService.AddFunctions(ctx, &req)
	if err != nil {
		logger.Errorf(c, "[ServiceTree API] 处理失败: %v", err)
		response.Internal(c, "处理失败: "+err.Error())
		return
	}

	response.OkWithData(c, resp)
}

// ExportCapabilityBundle 导出标准目录 JSON。
func (s *ServiceTree) ExportCapabilityBundle(c *gin.Context) {
	var req dto.ExportCapabilityBundleReq
	if c.Request.Method == http.MethodGet {
		if err := c.ShouldBindQuery(&req); err != nil {
			response.BadRequest(c, "参数错误: "+err.Error())
			return
		}
	} else if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	sourcePaths := req.SourceDirectoryPaths
	if req.SourceDirectoryPath != "" {
		sourcePaths = append(sourcePaths, req.SourceDirectoryPath)
	}
	if req.SourceRootPath != "" {
		sourcePaths = append(sourcePaths, req.SourceRootPath)
	}
	for _, sourcePath := range sourcePaths {
		if err := requireAccess(c, s.teamAccessService, sourcePath, access.ActionRead); err != nil {
			response.Error(c, err)
			return
		}
	}
	resp, err := s.serviceTreeService.ExportCapabilityBundle(ctx, &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OkWithData(c, resp)
}

// InstallCapabilityBundle 将目录 JSON 导入到目标目录节点下。
func (s *ServiceTree) InstallCapabilityBundle(c *gin.Context) {
	var req dto.InstallCapabilityBundleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	ctx := contextx.ToContext(c)
	if err := requireAccess(c, s.teamAccessService, req.TargetDirectoryPath, access.ActionAdmin); err != nil {
		response.Error(c, err)
		return
	}
	resp, err := s.serviceTreeService.InstallCapabilityBundle(ctx, &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OkWithDetailed(c, resp, resp.Message)
}
