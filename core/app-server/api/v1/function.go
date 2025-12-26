package v1

import (
	"strconv"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/service"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/enterprise"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/response"
	"github.com/ai-agent-os/ai-agent-os/pkg/license"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/pkg/middleware"
	"github.com/ai-agent-os/ai-agent-os/pkg/permission"
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
// @Description 根据函数ID获取函数的详细信息
// @Tags 函数管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param X-Token header string true "JWT Token"
// @Param function_id query int true "函数ID"
// @Success 200 {object} dto.GetFunctionResp "获取成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 401 {string} string "未授权"
// @Failure 404 {string} string "函数不存在"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/function/get [get]
func (f *Function) GetFunction(c *gin.Context) {
	var resp *dto.GetFunctionResp
	var err error

	// 从query参数获取函数ID
	functionIDStr := c.Query("function_id")
	if functionIDStr == "" {
		response.FailWithMessage(c, "缺少function_id参数")
		return
	}

	functionID, err := strconv.ParseInt(functionIDStr, 10, 64)
	if err != nil {
		response.FailWithMessage(c, "无效的函数ID")
		return
	}

	ctx := contextx.ToContext(c)

	// ⭐ 先获取函数信息，用于权限检查
	function, err := f.functionService.GetFunctionByID(ctx, functionID)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	// ⭐ 权限检查：检查是否有 function:read 权限
	// 使用函数的 Router 字段作为 full-code-path（Router 存储的就是 full-code-path）
	fullCodePath := function.Router
	if fullCodePath == "" {
		response.FailWithMessage(c, "函数路由信息不完整")
		return
	}

	// 检查权限功能是否启用（企业版）
	licenseMgr := license.GetManager()
	if licenseMgr.HasFeature(enterprise.FeaturePermission) {
		// 企业版：进行权限检查
		if !middleware.CheckPermissionWithPath(c, fullCodePath, "function:read", "无权限查看该函数详情") {
			return
		}
	}

	// 获取函数详情
	resp, err = f.functionService.GetFunction(ctx, functionID)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}

	// ⭐ 查询并返回函数权限信息（企业版功能）
	if licenseMgr.HasFeature(enterprise.FeaturePermission) {
		permissionService := enterprise.GetPermissionService()
		username := contextx.GetRequestUser(c)

		if permissionService != nil && username != "" && fullCodePath != "" {
			// 根据模板类型确定需要查询的权限点
			var actions []string
			actions = []string{
				"function:read",
				"function:execute",
			}
			// 根据模板类型添加操作级别权限
			switch resp.TemplateType {
			case "table":
				actions = append(actions, "table:create", "table:update", "table:delete")
			case "form":
				actions = append(actions, "form:submit")
			case "chart":
				// chart 使用 function:read 权限，拥有 read 权限即视为拥有 query 权限
			}

			if len(actions) > 0 {
				// ⭐ 支持权限继承：批量查询所有可能的权限点（当前资源 + 所有父目录的 directory:manage）
				resourcePaths := []string{fullCodePath}
				allActions := make([]string, len(actions))
				copy(allActions, actions)

				// 获取所有父目录路径
				parentPaths := permission.GetParentPaths(fullCodePath)
				for _, parentPath := range parentPaths {
					resourcePaths = append(resourcePaths, parentPath)
					allActions = append(allActions, "directory:manage")
				}

				permissions, err := permissionService.BatchCheckPermissions(ctx, username, resourcePaths, allActions)
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

					// 先检查当前资源的直接权限
					if nodePerms, ok := permissions[fullCodePath]; ok {
						for _, action := range actions {
							if hasPerm, ok := nodePerms[action]; ok {
								resp.Permissions[action] = hasPerm
							} else {
								resp.Permissions[action] = false
							}
						}
					} else {
						// 如果查询结果中没有该资源，初始化所有权限为 false
						for _, action := range actions {
							resp.Permissions[action] = false
						}
					}

					// ⭐ 如果直接权限都是 false，检查父目录的 directory:manage 权限（权限继承）
					hasAnyDirectPermission := false
					for _, action := range actions {
						if resp.Permissions[action] {
							hasAnyDirectPermission = true
							break
						}
					}

					if !hasAnyDirectPermission {
						// 检查父目录的 directory:manage 权限
						for _, parentPath := range parentPaths {
							if nodePerms, ok := permissions[parentPath]; ok {
								if hasManage, ok := nodePerms["directory:manage"]; ok && hasManage {
									// 父目录有管理权限，子资源自动拥有所有权限
									logger.Infof(c, "[Function API] 权限继承成功: resource=%s, parent=%s, has directory:manage",
										fullCodePath, parentPath)
									for _, action := range actions {
										resp.Permissions[action] = true
									}
									break
								}
							}
						}
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
