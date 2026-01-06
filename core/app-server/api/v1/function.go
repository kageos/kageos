package v1

import (
	"strconv"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/service"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/enterprise"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/response"
	"github.com/ai-agent-os/ai-agent-os/pkg/license"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/pkg/permission"
	permissionpkg "github.com/ai-agent-os/ai-agent-os/pkg/permission"
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
// @Description 根据 full-code-path 获取函数的详细信息
// @Tags 函数管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param full-code-path path string true "函数完整路径，如 /luobei/operations/crm/ticket"
// @Success 200 {object} dto.GetFunctionResp "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 403 {string} string "权限不足"
// @Failure 404 {string} string "函数不存在"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/function/info/*full-code-path [get]
func (f *Function) GetFunction(c *gin.Context) {
	// ⭐ 从路径参数获取 full-code-path
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

	// ⭐ 权限检查已由中间件自动处理（CheckFunctionRead），无需在此处再次检查

	// 获取函数详情
	resp, err := f.functionService.GetFunctionByFullCodePath(ctx, fullCodePath)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	// ⭐ 查询并返回函数权限信息（企业版功能）
	licenseMgr := license.GetManager()
	if licenseMgr.HasFeature(enterprise.FeaturePermission) {
		permissionService := enterprise.GetPermissionService()
		username := contextx.GetRequestUser(c)

		if permissionService != nil && username != "" && fullCodePath != "" {
			// ⭐ 统一权限点：所有函数类型统一使用 function:read/write/update/delete
			// ⭐ 优化：使用权限常量，避免硬编码
			actions := permission.FunctionActions

			if len(actions) > 0 {
				// ⭐ 使用 GetUserWorkspacePermissions 获取所有权限，然后在应用层校验
				// 从 resourcePath 解析 user 和 app
				_, user, app := permissionpkg.ParseFullCodePath(fullCodePath)
				if user != "" && app != "" {
					permReq := &enterprise.GetUserWorkspacePermissionsReq{
						User:           user,
						App:            app,
						Username:       username,
						DepartmentPath: contextx.GetRequestDepartmentFullPath(ctx),
					}
					
					permResp, err := permissionService.GetUserWorkspacePermissions(ctx, permReq)
					if err != nil {
						logger.Warnf(c, "[Function API] 查询权限失败: username=%s, resource=%s, error=%v",
							username, fullCodePath, err)
						// 权限查询失败，初始化所有权限为 false
						resp.Permissions = make(map[string]bool)
						for _, action := range actions {
							resp.Permissions[action] = false
						}
					} else {
						// 初始化权限 map
						resp.Permissions = make(map[string]bool)
						
						// ⭐ 使用响应对象的辅助方法检查每个权限（自动处理权限继承）
						for _, action := range actions {
							resp.Permissions[action] = permResp.CheckPermission(fullCodePath, action)
						}
					}
				} else {
					// 无法解析 user 和 app，初始化所有权限为 false
					resp.Permissions = make(map[string]bool)
					for _, action := range actions {
						resp.Permissions[action] = false
					}
				}
			}
		}
	}

	response.OkWithData(c, resp)
}

// GetFunctionsByApp 获取应用下所有函数
// @Summary 获取应用下所有函数
// @Description 根据应用ID获取该应用下的所有函数列表
// @Tags 函数管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param app_id query int true "应用ID"
// @Success 200 {object} dto.GetFunctionsByAppResp "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/function/list [get]
func (f *Function) GetFunctionsByApp(c *gin.Context) {
	var resp *dto.GetFunctionsByAppResp
	var err error

	// 从query参数获取应用ID
	appIDStr := c.Query("app_id")
	if appIDStr == "" {
		response.FailWithMessage(c, "缺少app_id参数")
		return
	}

	appID, err := strconv.ParseInt(appIDStr, 10, 64)
	if err != nil {
		response.FailWithMessage(c, "无效的应用ID")
		return
	}

	defer func() {
		logger.Infof(c, "GetFunctionsByApp app_id:%d resp:%+v err:%v", appID, resp, err)
	}()

	ctx := contextx.ToContext(c)
	resp, err = f.functionService.GetFunctionsByApp(ctx, appID)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	response.OkWithData(c, resp)
}
