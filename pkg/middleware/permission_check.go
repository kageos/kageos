package middleware

import (
	"context"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/enterprise"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/response"
	"github.com/ai-agent-os/ai-agent-os/pkg/license"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	permissionconstants "github.com/ai-agent-os/ai-agent-os/pkg/permission"
	"github.com/gin-gonic/gin"
)

// checkPermission 通用权限检查函数（内部使用）
// ⭐ 使用新的权限系统，自动支持权限继承（目录权限自动继承到子资源）
func checkPermission(c *gin.Context, action string, errorMessage string) bool {
	// 从 URL 路径参数提取 full-code-path
	fullCodePath := c.Param("full-code-path")
	if fullCodePath == "" {
		response.PermissionDenied(c, "无法获取资源路径", map[string]interface{}{
			"resource_path": "",
			"action":        action,
		})
		return false
	}

	// 确保路径以 / 开头
	if !strings.HasPrefix(fullCodePath, "/") {
		fullCodePath = "/" + fullCodePath
	}

	// ⭐ 运行时动态检查：根据当前 license 状态决定是否启用权限检查
	licenseMgr := license.GetManager()
	if !licenseMgr.HasFeature(enterprise.FeaturePermission) {
		// 社区版：不做权限控制，直接通过
		logger.Debugf(c, "[PermissionCheck] 社区版，跳过权限检查")
		return true
	}

	// 企业版：正常进行权限检查
	// 获取用户信息
	username := contextx.GetRequestUser(c)
	if username == "" {
		// ⭐ 添加调试日志，帮助排查用户信息丢失问题
		logger.Warnf(c, "[PermissionCheck] 用户信息为空 - FullCodePath: %s, Action: %s, X-Request-User Header: %s",
			fullCodePath, action, c.GetHeader("X-Request-User"))
		response.PermissionDenied(c, "未提供用户信息", map[string]interface{}{
			"resource_path": fullCodePath,
			"action":        action,
		})
		return false
	}

	// ⭐ 使用新的权限系统（直接调用 CheckPermission，内部已支持权限继承）
	permissionService := enterprise.GetPermissionService()
	ctx := contextx.ToContext(c)
	hasPermission, err := permissionService.CheckPermission(ctx, username, fullCodePath, action)
	if err != nil {
		permissionInfo := buildPermissionInfo(fullCodePath, action, "权限检查失败: "+err.Error())
		response.PermissionDenied(c, "权限检查失败: "+err.Error(), permissionInfo)
		return false
	}

	if !hasPermission {
		permissionInfo := buildPermissionInfo(fullCodePath, action, errorMessage)
		response.PermissionDenied(c, errorMessage, permissionInfo)
		return false
	}

	return true
}

// CheckTableSearch 检查表格查询权限（使用 table:read）
func CheckTableSearch() gin.HandlerFunc {
	return func(c *gin.Context) {
		action := permissionconstants.BuildActionCode(permissionconstants.ResourceTypeTable, "read")
		if !checkPermission(c, action, "无权限查看该表格") {
			return
		}
		c.Next()
	}
}

// CheckTableRead 检查表格读取权限（使用 table:read）
func CheckTableRead() gin.HandlerFunc {
	return func(c *gin.Context) {
		action := permissionconstants.BuildActionCode(permissionconstants.ResourceTypeTable, "read")
		if !checkPermission(c, action, "无权限查看该表格") {
			return
		}
		c.Next()
	}
}

// CheckTableWrite 检查表格写入权限（使用 table:write）
func CheckTableWrite() gin.HandlerFunc {
	return func(c *gin.Context) {
		action := permissionconstants.BuildActionCode(permissionconstants.ResourceTypeTable, "write")
		if !checkPermission(c, action, "无权限新增该表格记录") {
			return
		}
		c.Next()
	}
}

// CheckTableUpdate 检查表格更新权限（使用 table:update）
func CheckTableUpdate() gin.HandlerFunc {
	return func(c *gin.Context) {
		action := permissionconstants.BuildActionCode(permissionconstants.ResourceTypeTable, "update")
		if !checkPermission(c, action, "无权限更新该表格") {
			return
		}
		c.Next()
	}
}

// CheckTableDelete 检查表格删除权限（使用 table:delete）
func CheckTableDelete() gin.HandlerFunc {
	return func(c *gin.Context) {
		action := permissionconstants.BuildActionCode(permissionconstants.ResourceTypeTable, "delete")
		if !checkPermission(c, action, "无权限删除该表格") {
			return
		}
		c.Next()
	}
}

// CheckFormWrite 检查表单写入权限（使用 form:write）
func CheckFormWrite() gin.HandlerFunc {
	return func(c *gin.Context) {
		action := permissionconstants.BuildActionCode(permissionconstants.ResourceTypeForm, "write")
		if !checkPermission(c, action, "无权限提交该表单") {
			return
		}
		c.Next()
	}
}

// CheckChartQuery 检查图表查询权限（使用 chart:read）
func CheckChartQuery() gin.HandlerFunc {
	return func(c *gin.Context) {
		action := permissionconstants.BuildActionCode(permissionconstants.ResourceTypeChart, "read")
		if !checkPermission(c, action, "无权限查看该图表") {
			return
		}
		c.Next()
	}
}

// CheckFunctionRead 检查函数读取权限（根据函数类型动态确定权限点：table:read、form:read、chart:read）
// ⭐ 函数类型直接从 URL 路径参数获取（/info/:func-type/*full-code-path），无需查询数据库
func CheckFunctionRead() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从 URL 路径参数提取函数类型和 full-code-path
		funcType := c.Param("func-type")
		fullCodePath := c.Param("full-code-path")
		
		if fullCodePath == "" {
			response.PermissionDenied(c, "无法获取资源路径", map[string]interface{}{
				"resource_path": "",
				"action":        "",
			})
			return
		}

		// 确保路径以 / 开头
		if !strings.HasPrefix(fullCodePath, "/") {
			fullCodePath = "/" + fullCodePath
		}

		// ⭐ 运行时动态检查：根据当前 license 状态决定是否启用权限检查
		licenseMgr := license.GetManager()
		if !licenseMgr.HasFeature(enterprise.FeaturePermission) {
			// 社区版：不做权限控制，直接通过
			logger.Debugf(c, "[PermissionCheck] 社区版，跳过权限检查")
			c.Next()
			return
		}

		// 企业版：正常进行权限检查
		// 获取用户信息
		username := contextx.GetRequestUser(c)
		if username == "" {
			response.PermissionDenied(c, "未提供用户信息", map[string]interface{}{
				"resource_path": fullCodePath,
				"action":        "",
			})
			return
		}

		// ⭐ 根据函数类型直接构造权限点（无需查询数据库）
		var action string
		var errorMessage string

		// 根据函数类型确定资源类型和权限点
		resourceType := permissionconstants.GetResourceType("function", funcType)
		if resourceType != "" {
			action = permissionconstants.BuildActionCode(resourceType, "read")
			errorMessage = "无权限查看该函数详情"
		} else {
			// 如果函数类型无效，使用默认的 table:read（兼容旧逻辑）
			action = permissionconstants.BuildActionCode(permissionconstants.ResourceTypeTable, "read")
			errorMessage = "无权限查看该函数详情"
		}

		// ⭐ 使用新的权限系统（直接调用 CheckPermission，内部已支持权限继承）
		permissionService := enterprise.GetPermissionService()
		ctx := contextx.ToContext(c)
		hasPermission, err := permissionService.CheckPermission(ctx, username, fullCodePath, action)
		if err != nil {
			permissionInfo := buildPermissionInfo(fullCodePath, action, "权限检查失败: "+err.Error())
			response.PermissionDenied(c, "权限检查失败: "+err.Error(), permissionInfo)
			return
		}

		if !hasPermission {
			permissionInfo := buildPermissionInfo(fullCodePath, action, errorMessage)
			response.PermissionDenied(c, errorMessage, permissionInfo)
			return
		}

		c.Next()
	}
}

// CheckAppUpdate 检查应用更新权限
func CheckAppUpdate() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从路径参数获取应用信息，构建 full-code-path
		app := c.Param("app")
		user := contextx.GetRequestUser(c)
		if user == "" || app == "" {
			response.PermissionDenied(c, "无法获取用户信息或应用信息", map[string]interface{}{
				"resource_path": "",
				"action":        "app:update",
			})
			return
		}
		fullCodePath := "/" + user + "/" + app
		action := permissionconstants.BuildActionCode(permissionconstants.ResourceTypeApp, permissionconstants.ActionUpdate)
		if !checkPermissionForPath(c, fullCodePath, action, "无权限更新该应用") {
			return
		}
		c.Next()
	}
}

// CheckAppDelete 检查应用删除权限
func CheckAppDelete() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从路径参数获取应用信息，构建 full-code-path
		app := c.Param("app")
		user := contextx.GetRequestUser(c)
		if user == "" || app == "" {
			response.PermissionDenied(c, "无法获取用户信息或应用信息", map[string]interface{}{
				"resource_path": "",
				"action":        "app:delete",
			})
			return
		}
		fullCodePath := "/" + user + "/" + app
		action := permissionconstants.BuildActionCode(permissionconstants.ResourceTypeApp, permissionconstants.ActionDelete)
		if !checkPermissionForPath(c, fullCodePath, action, "无权限删除该应用") {
			return
		}
		c.Next()
	}
}

// CheckWorkspaceUpdate 检查工作空间更新权限（需要 app:admin 权限）
func CheckWorkspaceUpdate() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从路径参数获取租户和应用信息
		user := c.Param("user")
		app := c.Param("app")
		if user == "" || app == "" {
			response.PermissionDenied(c, "无法获取租户或应用信息", map[string]interface{}{
				"resource_path": "",
				"action":        "app:admin",
			})
			return
		}
		
		// 构建 full-code-path
		fullCodePath := "/" + user + "/" + app
		
		// 检查是否有 app:admin 权限
		actionCode := permissionconstants.BuildActionCode(permissionconstants.ResourceTypeApp, permissionconstants.ActionAdmin)
		if !checkPermissionForPath(c, fullCodePath, actionCode, "无权限更新该工作空间") {
			return
		}
		c.Next()
	}
}

// checkPermissionForPath 检查指定路径的权限（内部使用）
func checkPermissionForPath(c *gin.Context, fullCodePath string, action string, errorMessage string) bool {
	// ⭐ 运行时动态检查：根据当前 license 状态决定是否启用权限检查
	licenseMgr := license.GetManager()
	if !licenseMgr.HasFeature(enterprise.FeaturePermission) {
		// 社区版：不做权限控制，直接通过
		logger.Debugf(c, "[PermissionCheck] 社区版，跳过权限检查")
		return true
	}

	// 企业版：正常进行权限检查
	if fullCodePath == "" {
		response.PermissionDenied(c, "资源路径不能为空", map[string]interface{}{
			"resource_path": "",
			"action":        action,
		})
		return false
	}

	// 获取用户信息
	username := contextx.GetRequestUser(c)
	if username == "" {
		response.PermissionDenied(c, "未提供用户信息", map[string]interface{}{
			"resource_path": fullCodePath,
			"action":        action,
		})
		return false
	}

	// ⭐ 使用新的权限系统（直接调用 CheckPermission，内部已支持权限继承）
	permissionService := enterprise.GetPermissionService()
	ctx := contextx.ToContext(c)
	hasPermission, err := permissionService.CheckPermission(ctx, username, fullCodePath, action)
	if err != nil {
		permissionInfo := buildPermissionInfo(fullCodePath, action, "权限检查失败: "+err.Error())
		response.PermissionDenied(c, "权限检查失败: "+err.Error(), permissionInfo)
		return false
	}

	if !hasPermission {
		permissionInfo := buildPermissionInfo(fullCodePath, action, errorMessage)
		response.PermissionDenied(c, errorMessage, permissionInfo)
		return false
	}

	return true
}

// buildPermissionInfo 构建权限详细信息，方便前端构造申请权限的提示
func buildPermissionInfo(resourcePath string, action string, errorMessage string) map[string]interface{} {
	// 获取操作显示名称
	actionDisplay := getActionDisplayName(action)

	// 构建申请权限的 URL（前端可以根据这个 URL 跳转到权限申请页面）
	applyURL := buildPermissionApplyURL(resourcePath, action)

	return map[string]interface{}{
		"resource_path":  resourcePath,  // 资源路径
		"action":         action,        // 权限点（如 function:read）
		"action_display": actionDisplay, // 操作显示名称（如 "表格查询"）
		"apply_url":      applyURL,      // 申请权限的 URL（前端可以直接使用）
		"error_message":  errorMessage,  // 错误消息
	}
}

// getActionDisplayName 获取操作显示名称
func getActionDisplayName(action string) string {
	// ⭐ 使用权限点编码（resource_type:action_type）作为 key，避免重复
	displayNames := map[string]string{
		// Table 函数操作
		permissionconstants.BuildActionCode(permissionconstants.ResourceTypeTable, permissionconstants.ActionRead):   "表格查看",
		permissionconstants.BuildActionCode(permissionconstants.ResourceTypeTable, permissionconstants.ActionWrite):  "表格写入",
		permissionconstants.BuildActionCode(permissionconstants.ResourceTypeTable, permissionconstants.ActionUpdate): "表格更新",
		permissionconstants.BuildActionCode(permissionconstants.ResourceTypeTable, permissionconstants.ActionDelete): "表格删除",
		permissionconstants.BuildActionCode(permissionconstants.ResourceTypeTable, permissionconstants.ActionAdmin): "表格管理",
		// Form 函数操作
		permissionconstants.BuildActionCode(permissionconstants.ResourceTypeForm, permissionconstants.ActionRead):   "表单查看",
		permissionconstants.BuildActionCode(permissionconstants.ResourceTypeForm, permissionconstants.ActionWrite):  "表单提交",
		permissionconstants.BuildActionCode(permissionconstants.ResourceTypeForm, permissionconstants.ActionAdmin): "表单管理",
		// Chart 函数操作
		permissionconstants.BuildActionCode(permissionconstants.ResourceTypeChart, permissionconstants.ActionRead):   "图表查看",
		permissionconstants.BuildActionCode(permissionconstants.ResourceTypeChart, permissionconstants.ActionAdmin): "图表管理",
		// Directory 操作
		permissionconstants.BuildActionCode(permissionconstants.ResourceTypeDirectory, permissionconstants.ActionRead):   "目录查看",
		permissionconstants.BuildActionCode(permissionconstants.ResourceTypeDirectory, permissionconstants.ActionWrite):  "目录写入",
		permissionconstants.BuildActionCode(permissionconstants.ResourceTypeDirectory, permissionconstants.ActionUpdate): "目录更新",
		permissionconstants.BuildActionCode(permissionconstants.ResourceTypeDirectory, permissionconstants.ActionDelete): "目录删除",
		permissionconstants.BuildActionCode(permissionconstants.ResourceTypeDirectory, permissionconstants.ActionAdmin): "目录管理",
		// App 操作
		permissionconstants.BuildActionCode(permissionconstants.ResourceTypeApp, permissionconstants.ActionRead):   "工作空间查看",
		permissionconstants.BuildActionCode(permissionconstants.ResourceTypeApp, permissionconstants.ActionWrite):  "工作空间创建",
		permissionconstants.BuildActionCode(permissionconstants.ResourceTypeApp, permissionconstants.ActionUpdate): "工作空间更新",
		permissionconstants.BuildActionCode(permissionconstants.ResourceTypeApp, permissionconstants.ActionDelete): "工作空间删除",
		permissionconstants.BuildActionCode(permissionconstants.ResourceTypeApp, permissionconstants.ActionAdmin): "工作空间管理",
	}

	if displayName, ok := displayNames[action]; ok {
		return displayName
	}

	// 如果没有找到，返回原始 action
	return action
}

// CheckFunctionExecute 检查函数执行权限（动态根据函数类型和HTTP方法确定权限点）
// ⭐ 用于 /api/v1/run/*router 路由
func CheckFunctionExecute(getFunctionDetail func(ctx context.Context, fullCodePath string) (templateType string, err error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从 URL 路径参数提取 full-code-path
		fullCodePath := c.Param("router")
		if fullCodePath == "" {
			response.PermissionDenied(c, "无法获取资源路径", map[string]interface{}{
				"resource_path": "",
				"action":        "",
			})
			return
		}

		// 确保路径以 / 开头
		if !strings.HasPrefix(fullCodePath, "/") {
			fullCodePath = "/" + fullCodePath
		}

		// ⭐ 运行时动态检查：根据当前 license 状态决定是否启用权限检查
		licenseMgr := license.GetManager()
		if !licenseMgr.HasFeature(enterprise.FeaturePermission) {
			// 社区版：不做权限控制，直接通过
			logger.Debugf(c, "[PermissionCheck] 社区版，跳过权限检查")
			c.Next()
			return
		}

		// 企业版：正常进行权限检查
		// 获取用户信息
		username := contextx.GetRequestUser(c)
		if username == "" {
			response.PermissionDenied(c, "未提供用户信息", map[string]interface{}{
				"resource_path": fullCodePath,
				"action":        "",
			})
			return
		}

		// 根据函数类型和HTTP方法，动态确定权限点
		var action string
		var errorMessage string

		// 尝试获取函数详情（如果提供了函数详情获取函数）
		if getFunctionDetail != nil {
			ctx := contextx.ToContext(c)
			templateType, err := getFunctionDetail(ctx, fullCodePath)
			if err == nil {
				// 根据模板类型和HTTP方法确定权限点
				action, errorMessage = determinePermissionAction(templateType, c.Request.Method)
			}
		}

		// 如果无法确定权限点，使用默认的 table:admin 权限（所有权）
		if action == "" {
			action = permissionconstants.BuildActionCode(permissionconstants.ResourceTypeTable, permissionconstants.ActionAdmin)
			errorMessage = "无权限执行该函数"
		}

		// ⭐ 使用新的权限系统（直接调用 CheckPermission，内部已支持权限继承）
		permissionService := enterprise.GetPermissionService()
		ctx := contextx.ToContext(c)
		hasPermission, err := permissionService.CheckPermission(ctx, username, fullCodePath, action)
		if err != nil {
			permissionInfo := buildPermissionInfo(fullCodePath, action, "权限检查失败: "+err.Error())
			response.PermissionDenied(c, "权限检查失败: "+err.Error(), permissionInfo)
			return
		}

		if !hasPermission {
			permissionInfo := buildPermissionInfo(fullCodePath, action, errorMessage)
			response.PermissionDenied(c, errorMessage, permissionInfo)
			return
		}

		c.Next()
	}
}

// determinePermissionAction 根据模板类型和HTTP方法确定权限点和错误消息
func determinePermissionAction(templateType string, httpMethod string) (action string, errorMessage string) {
	// 根据模板类型确定资源类型
	resourceType := permissionconstants.GetResourceType("function", templateType)
	if resourceType == "" {
		resourceType = permissionconstants.ResourceTypeTable // 默认使用 table
	}

	switch httpMethod {
	case "GET":
		// 所有类型的查询都使用 read
		action = permissionconstants.BuildActionCode(resourceType, permissionconstants.ActionRead)
		switch templateType {
		case "table":
			return action, "无权限查看该表格"
		case "chart":
			return action, "无权限查看该图表"
		default:
			return action, "无权限查看该函数"
		}
	case "POST":
		// 所有类型的创建/提交都使用 write
		action = permissionconstants.BuildActionCode(resourceType, permissionconstants.ActionWrite)
		switch templateType {
		case "table":
			return action, "无权限新增该表格记录"
		case "form":
			return action, "无权限提交该表单"
		default:
			return action, "无权限执行该操作"
		}
	case "PUT", "PATCH":
		// 所有类型的更新都使用 update
		action = permissionconstants.BuildActionCode(resourceType, permissionconstants.ActionUpdate)
		switch templateType {
		case "table":
			return action, "无权限更新该表格"
		default:
			return action, "无权限更新该函数"
		}
	case "DELETE":
		// 所有类型的删除都使用 delete
		action = permissionconstants.BuildActionCode(resourceType, permissionconstants.ActionDelete)
		switch templateType {
		case "table":
			return action, "无权限删除该表格"
		default:
			return action, "无权限删除该函数"
		}
	default:
		// 其他方法：使用 admin（所有权）
		action = permissionconstants.BuildActionCode(resourceType, permissionconstants.ActionAdmin)
		return action, "无权限执行该函数"
	}
}

// buildPermissionApplyURL 构建申请权限的 URL
func buildPermissionApplyURL(resourcePath string, action string) string {
	// 前端可以根据这个 URL 跳转到权限申请页面
	// 格式：/permissions/apply?resource={resourcePath}&action={action}
	return "/permissions/apply?resource=" + resourcePath + "&action=" + action
}
