package middleware

import (
	"context"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/enterprise"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/response"
	"github.com/ai-agent-os/ai-agent-os/pkg/license"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/gin-gonic/gin"
)

// checkPermission 通用权限检查函数（内部使用，支持企业版控制）
// 从 URL 中自动提取 full-code-path
func checkPermission(c *gin.Context, action string, errorMessage string) bool {
	// 从 URL 中提取 full-code-path
	fullCodePath := extractFullCodePathFromURL(c.Request.URL.Path)
	if fullCodePath == "" {
		response.PermissionDenied(c, "无法从 URL 中提取资源路径", map[string]interface{}{
			"resource_path": "",
			"action":        action,
		})
		return false
	}

	return checkPermissionWithPath(c, fullCodePath, action, errorMessage)
}

// CheckPermissionWithPath 通用权限检查函数（导出，供其他包使用）
// 使用指定的 full-code-path
func CheckPermissionWithPath(c *gin.Context, fullCodePath string, action string, errorMessage string) bool {
	return checkPermissionWithPath(c, fullCodePath, action, errorMessage)
}

// checkPermissionWithPath 通用权限检查函数（内部使用，支持企业版控制）
// 使用指定的 full-code-path
func checkPermissionWithPath(c *gin.Context, fullCodePath string, action string, errorMessage string) bool {
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

	// 检查权限（使用 enterprise 接口）
	permissionService := enterprise.GetPermissionService()
	ctx := contextx.ToContext(c)
	hasPermission, err := permissionService.CheckPermission(ctx, username, fullCodePath, action)
	if err != nil {
		permissionInfo := buildPermissionInfo(fullCodePath, action, "权限检查失败: "+err.Error())
		response.PermissionDenied(c, "权限检查失败: "+err.Error(), permissionInfo)
		return false
	}

	if !hasPermission {
		// 构建权限详细信息，方便前端构造申请权限的提示
		permissionInfo := buildPermissionInfo(fullCodePath, action, errorMessage)
		response.PermissionDenied(c, errorMessage, permissionInfo)
		return false
	}

	return true
}

// checkPermissionDynamic 动态权限检查函数（根据函数类型和HTTP方法自动确定权限点）
func checkPermissionDynamic(c *gin.Context, getFunctionDetail func(ctx context.Context, fullCodePath string) (templateType string, err error)) bool {
	// ⭐ 运行时动态检查：根据当前 license 状态决定是否启用权限检查
	licenseMgr := license.GetManager()
	if !licenseMgr.HasFeature(enterprise.FeaturePermission) {
		// 社区版：不做权限控制，直接通过
		logger.Debugf(c, "[PermissionCheck] 社区版，跳过权限检查")
		return true
	}

	// 企业版：正常进行权限检查
	// 从 URL 中提取 full-code-path
	fullCodePath := extractFullCodePathFromURL(c.Request.URL.Path)
	if fullCodePath == "" {
		response.PermissionDenied(c, "无法从 URL 中提取资源路径", map[string]interface{}{
			"resource_path": "",
			"action":        "",
		})
		return false
	}

	// 获取用户信息
	username := contextx.GetRequestUser(c)
	if username == "" {
		response.PermissionDenied(c, "未提供用户信息", map[string]interface{}{
			"resource_path": fullCodePath,
			"action":        "",
		})
		return false
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

	// 如果无法确定权限点，使用默认的 execute 权限
	if action == "" {
		action = "function:execute"
		errorMessage = "无权限执行该函数"
	}

	// 检查权限（使用 enterprise 接口）
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

// determinePermissionAction 根据模板类型和HTTP方法确定权限点和错误消息
func determinePermissionAction(templateType string, httpMethod string) (action string, errorMessage string) {
	switch templateType {
	case "table":
		switch httpMethod {
		case "GET":
			return "table:search", "无权限查询该表格"
		case "POST":
			return "table:create", "无权限创建该表格"
		case "PUT", "PATCH":
			return "table:update", "无权限更新该表格"
		case "DELETE":
			return "table:delete", "无权限删除该表格"
		default:
			return "function:execute", "无权限执行该函数"
		}
	case "form":
		switch httpMethod {
		case "POST", "PUT", "PATCH":
			return "form:submit", "无权限提交该表单"
		default:
			return "function:execute", "无权限执行该函数"
		}
	case "chart":
		switch httpMethod {
		case "GET", "POST":
			return "chart:query", "无权限查询该图表"
		default:
			return "function:execute", "无权限执行该函数"
		}
	default:
		// 其他类型或未指定：使用 function:execute
		return "function:execute", "无权限执行该函数"
	}
}

// CheckTableSearch 检查表格查询权限
func CheckTableSearch() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !checkPermission(c, "table:search", "无权限查询该表格") {
			return
		}
		c.Next()
	}
}

// CheckTableCreate 检查表格创建权限
func CheckTableCreate() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !checkPermission(c, "table:create", "无权限创建该表格") {
			return
		}
		c.Next()
	}
}

// CheckTableUpdate 检查表格更新权限
func CheckTableUpdate() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !checkPermission(c, "table:update", "无权限更新该表格") {
			return
		}
		c.Next()
	}
}

// CheckTableDelete 检查表格删除权限
func CheckTableDelete() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !checkPermission(c, "table:delete", "无权限删除该表格") {
			return
		}
		c.Next()
	}
}

// CheckFormSubmit 检查表单提交权限
func CheckFormSubmit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !checkPermission(c, "form:submit", "无权限提交该表单") {
			return
		}
		c.Next()
	}
}

// CheckChartQuery 检查图表查询权限
func CheckChartQuery() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !checkPermission(c, "chart:query", "无权限查询该图表") {
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
		if !checkPermissionWithPath(c, fullCodePath, "app:update", "无权限更新该应用") {
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
		if !checkPermissionWithPath(c, fullCodePath, "app:delete", "无权限删除该应用") {
			return
		}
		c.Next()
	}
}

// CheckCallback 检查回调接口权限
func CheckCallback() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 回调接口统一使用 callback:on_select_fuzzy 权限（或根据 _type 参数动态确定）
		callbackType := c.Query("_type")
		action := "callback:on_select_fuzzy"
		errorMessage := "无权限调用该回调接口"

		// 可以根据 callbackType 进一步细化权限点（如 OnTableAddRow、OnTableUpdateRow 等）
		if callbackType != "" {
			switch callbackType {
			case "OnTableAddRow":
				action = "table:create"
				errorMessage = "无权限新增表格数据"
			case "OnTableUpdateRow":
				action = "table:update"
				errorMessage = "无权限更新表格数据"
			case "OnTableDeleteRows":
				action = "table:delete"
				errorMessage = "无权限删除表格数据"
			case "OnSelectFuzzy":
				action = "callback:on_select_fuzzy"
				errorMessage = "无权限调用模糊选择回调"
			}
		}

		if !checkPermission(c, action, errorMessage) {
			return
		}
		c.Next()
	}
}

// CheckFunctionExecute 检查函数执行权限（动态根据函数类型和HTTP方法确定权限点）
// 注意：这个中间件需要能够获取函数详情（template_type），可能需要查询数据库
// 如果无法获取函数详情，则使用默认的 function:execute 权限
func CheckFunctionExecute(getFunctionDetail func(ctx context.Context, fullCodePath string) (templateType string, err error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !checkPermissionDynamic(c, getFunctionDetail) {
			return
		}
		c.Next()
	}
}

// ============================================
// 辅助函数
// ============================================

// extractFullCodePathFromURL 从 URL 中提取 full-code-path
func extractFullCodePathFromURL(urlPath string) string {
	// 移除 /workspace/api/v1 前缀
	urlPath = strings.TrimPrefix(urlPath, "/workspace/api/v1")
	
	// 移除操作前缀（如 /table/search、/form/submit、/chart/query、/callback/on_select_fuzzy、/run、/callback 等）
	urlPath = strings.TrimPrefix(urlPath, "/table/search")
	urlPath = strings.TrimPrefix(urlPath, "/table/create")
	urlPath = strings.TrimPrefix(urlPath, "/table/update")
	urlPath = strings.TrimPrefix(urlPath, "/table/delete")
	urlPath = strings.TrimPrefix(urlPath, "/form/submit")
	urlPath = strings.TrimPrefix(urlPath, "/chart/query")
	urlPath = strings.TrimPrefix(urlPath, "/callback/on_select_fuzzy")
	urlPath = strings.TrimPrefix(urlPath, "/run")
	urlPath = strings.TrimPrefix(urlPath, "/callback")

	// 移除开头的斜杠
	urlPath = strings.TrimPrefix(urlPath, "/")

	// 如果路径为空，返回空
	if urlPath == "" {
		return ""
	}

	// 构建 full-code-path（必须以 / 开头）
	return "/" + urlPath
}

// isValidTenantName 验证租户名称格式
func isValidTenantName(name string) bool {
	if len(name) == 0 {
		return false
	}
	for _, char := range name {
		if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '_') {
			return false
		}
	}
	return true
}

// isValidAppName 验证应用名称格式
func isValidAppName(name string) bool {
	return isValidTenantName(name)
}

// buildPermissionInfo 构建权限详细信息，方便前端构造申请权限的提示
func buildPermissionInfo(resourcePath string, action string, errorMessage string) map[string]interface{} {
	// 获取操作显示名称
	actionDisplay := getActionDisplayName(action)

	// 构建申请权限的 URL（前端可以根据这个 URL 跳转到权限申请页面）
	applyURL := buildPermissionApplyURL(resourcePath, action)

	return map[string]interface{}{
		"resource_path":  resourcePath,              // 资源路径
		"action":         action,                    // 权限点（如 table:search）
		"action_display": actionDisplay,             // 操作显示名称（如 "表格查询"）
		"apply_url":       applyURL,                 // 申请权限的 URL（前端可以直接使用）
		"error_message":   errorMessage,            // 错误消息
	}
}

// getActionDisplayName 获取操作显示名称
func getActionDisplayName(action string) string {
	displayNames := map[string]string{
		// Table 操作
		"table:search": "表格查询",
		"table:create": "表格新增",
		"table:update": "表格更新",
		"table:delete": "表格删除",
		// Form 操作
		"form:submit": "表单提交",
		// Chart 操作
		"chart:query": "图表查询",
		// Callback 操作
		"callback:on_select_fuzzy": "模糊搜索回调",
		// Function 操作
		"function:read":    "函数查看",
		"function:execute": "函数执行",
		// Directory 操作
		"directory:read":    "目录查看",
		"directory:create":  "目录创建",
		"directory:update":  "目录更新",
		"directory:delete":  "目录删除",
		"directory:manage": "目录管理",
		// App 操作
		"app:read":   "应用查看",
		"app:create": "应用创建",
		"app:update": "应用更新",
		"app:delete": "应用删除",
		"app:manage": "应用管理",
		"app:deploy": "应用部署",
	}

	if displayName, ok := displayNames[action]; ok {
		return displayName
	}

	// 如果没有找到，返回原始 action
	return action
}

// buildPermissionApplyURL 构建申请权限的 URL
func buildPermissionApplyURL(resourcePath string, action string) string {
	// 前端可以根据这个 URL 跳转到权限申请页面
	// 格式：/permissions/apply?resource={resourcePath}&action={action}
	return "/permissions/apply?resource=" + resourcePath + "&action=" + action
}

