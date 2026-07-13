package v1

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/pkg/access"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/ginx/response"
)

// ============================================
// Callback 接口
// ============================================

// CallbackOnSelectFuzzy 模糊搜索回调接口
// @Summary 模糊搜索回调
// @Description Select 组件的模糊搜索回调
// @Tags 标准接口
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param full-code-path path string true "函数完整路径，如：/luobei/operations/tools/pdftools/to_images"
// @Param body body object true "搜索条件，格式：{\"code\": \"field_code\", \"type\": \"by_values\", \"value\": [1, 2, 3], \"request\": {...}}"
// @Success 200 {object} dto.RequestAppResp "查询成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 403 {string} string "权限不足"
// @Failure 500 {string} string "服务器内部错误"
// @Router /workspace/api/v1/selection-options/{full-code-path} [post]
func (s *StandardAPI) CallbackOnSelectFuzzy(c *gin.Context) {
	fullCodePath := normalizeFullCodePathParam(c)
	if fullCodePath == "" {
		response.BadRequest(c, "full-code-path 参数不能为空")
		return
	}
	if err := requireWorkspaceDataAccess(c, s.teamAccessService, fullCodePath, access.ActionRead); err != nil {
		response.Error(c, err)
		return
	}

	// 构建回调请求对象（调用 OnSelectFuzzy）
	req, err := s.buildCallbackAppReq(c, fullCodePath, "OnSelectFuzzy")
	if err != nil {
		response.Internal(c, "构建请求失败: "+err.Error())
		return
	}

	// 调用服务层
	ctx := contextx.ToContext(c)
	now := time.Now()
	resp, err := s.appService.RequestApp(ctx, req)
	mill := time.Since(now).Milliseconds()

	// 构建响应元数据
	metadata := make(map[string]interface{})
	metadata["trace_id"] = req.TraceId
	metadata["app"] = req.App
	if resp != nil {
		metadata["version"] = resp.Version
	}
	metadata["total_cost_mill"] = mill

	if err != nil {
		response.Error(c, err)
		return
	}

	if resp.Error != "" {
		response.ApplicationError(c, resp.ErrCode, resp.Error, metadata)
		return
	}

	response.OkWithData(c, resp.Result, metadata)
}
