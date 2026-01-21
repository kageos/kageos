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
// @Router /api/v1/function/info/{func-type}/*full-code-path [get]
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
			// ⭐ 获取节点需要的权限点（格式：resource_type:action_type，如 table:read, form:write）
			// 从 resourcePath 解析 user 和 app
			_, user, app := permissionpkg.ParseFullCodePath(fullCodePath)
			if user != "" && app != "" {
				// ⭐ 优先检查：如果当前用户是工作空间管理员，直接返回所有权限
				// 获取应用信息（检查 admins 字段）
				appModel, err := f.functionService.GetAppByUserAndCode(ctx, user, app)
				if err == nil && appModel != nil && appModel.Admins != "" {
					adminList := strings.Split(appModel.Admins, ",")
					for _, admin := range adminList {
						admin = strings.TrimSpace(admin)
						if admin == username {
							// 当前用户是管理员，直接返回所有权限
							logger.Debugf(c, "[Function API] 用户 %s 是工作空间管理员，直接返回所有权限", username)
							// 获取函数类型（从函数详情中获取 templateType）
							templateType := ""
							if resp != nil && resp.TemplateType != "" {
								templateType = resp.TemplateType
							}
							actions := permission.GetActionsForNode("function", templateType)
							resp.Permissions = make(map[string]bool)
							for _, actionCode := range actions {
								resp.Permissions[actionCode] = true
							}
							// ⭐ 同时将 app:admin 权限添加到节点权限中，方便前端检查
							appAdminCode := permission.BuildActionCode(permission.ResourceTypeApp, "admin")
							resp.Permissions[appAdminCode] = true
							response.OkWithData(c, resp)
							return
						}
					}
				}

				// ⭐ 如果不是管理员，使用原有的权限查询逻辑
				// 获取函数类型（需要从函数详情中获取 templateType）
				// 这里暂时使用默认的 table 类型，实际应该从函数详情中获取
				templateType := "" // 如果无法获取，GetActionsForNode 会使用默认值
				if resp != nil && resp.TemplateType != "" {
					templateType = resp.TemplateType
				}
				actions := permission.GetActionsForNode("function", templateType)

				if len(actions) > 0 {
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
						for _, actionCode := range actions {
							resp.Permissions[actionCode] = false
						}
					} else {
						// 初始化权限 map
						resp.Permissions = make(map[string]bool)

						// ⭐ 使用响应对象的辅助方法检查每个权限（自动处理权限继承）
						// 权限点格式：resource_type:action_type
						for _, actionCode := range actions {
							resp.Permissions[actionCode] = permResp.CheckPermission(fullCodePath, actionCode)
						}
					}
				} else {
					// 无法获取权限点，初始化所有权限为 false
					resp.Permissions = make(map[string]bool)
				}
			} else {
				// 无法解析 user 和 app，初始化所有权限为 false
				resp.Permissions = make(map[string]bool)
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
