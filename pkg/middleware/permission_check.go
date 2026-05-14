package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"strconv"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/enterprise"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/response"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	permissionconstants "github.com/ai-agent-os/ai-agent-os/pkg/permission"
	"github.com/gin-gonic/gin"
)

func skipPermissionChecks(c *gin.Context) bool {
	if permissionconstants.EnforcementEnabled() {
		return false
	}
	c.Next()
	return true
}

// checkPermissionWithPath 通用权限检查（显式传入资源路径，供帖子等从 query/body 取 path 的场景使用）
func checkPermissionWithPath(c *gin.Context, fullCodePath string, action string, errorMessage string) bool {
	if !permissionconstants.EnforcementEnabled() {
		return true
	}
	fullCodePath = strings.TrimSpace(fullCodePath)
	if fullCodePath == "" {
		response.PermissionDenied(c, "无法获取资源路径", map[string]interface{}{
			"resource_path": "",
			"action":        action,
		})
		return false
	}
	if !strings.HasPrefix(fullCodePath, "/") {
		fullCodePath = "/" + fullCodePath
	}
	username := contextx.GetRequestUser(c)
	if username == "" {
		logger.Warnf(c, "[PermissionCheck] 用户信息为空 - FullCodePath: %s, Action: %s", fullCodePath, action)
		response.PermissionDenied(c, "未提供用户信息", map[string]interface{}{
			"resource_path": fullCodePath,
			"action":        action,
		})
		return false
	}
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

// checkPermission 通用权限检查函数（从 URL 参数 full-code-path 取路径）
// ⭐ 使用新的权限系统，自动支持权限继承（目录权限自动继承到子资源）
func checkPermission(c *gin.Context, action string, errorMessage string) bool {
	fullCodePath := c.Param("full-code-path")
	if fullCodePath == "" {
		response.PermissionDenied(c, "无法获取资源路径", map[string]interface{}{
			"resource_path": "",
			"action":        action,
		})
		return false
	}
	return checkPermissionWithPath(c, fullCodePath, action, errorMessage)
}

// CheckTableSearch 检查表格查询权限（使用 table:read）
func CheckTableSearch() gin.HandlerFunc {
	return func(c *gin.Context) {
		if skipPermissionChecks(c) {
			return
		}
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
		if skipPermissionChecks(c) {
			return
		}
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
		if skipPermissionChecks(c) {
			return
		}
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
		if skipPermissionChecks(c) {
			return
		}
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
		if skipPermissionChecks(c) {
			return
		}
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
		if skipPermissionChecks(c) {
			return
		}
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
		if skipPermissionChecks(c) {
			return
		}
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
		if skipPermissionChecks(c) {
			return
		}
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
		if skipPermissionChecks(c) {
			return
		}
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
		if skipPermissionChecks(c) {
			return
		}
		fullCodePath := strings.TrimSpace(c.Query("resource_path"))
		if fullCodePath == "" {
			// 兼容旧路由：从路径参数获取应用信息，构建 full-code-path
			app := c.Param("app")
			user := contextx.GetRequestUser(c)
			if user != "" && app != "" {
				fullCodePath = "/" + user + "/" + app
			}
		}
		if fullCodePath == "" {
			response.PermissionDenied(c, "无法获取用户信息或应用信息", map[string]interface{}{
				"resource_path": "",
				"action":        "app:delete",
			})
			return
		}
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
		if skipPermissionChecks(c) {
			return
		}
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
	if !permissionconstants.EnforcementEnabled() {
		return true
	}
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
		permissionconstants.BuildActionCode(permissionconstants.ResourceTypeTable, permissionconstants.ActionAdmin):  "表格管理",
		// Form 函数操作
		permissionconstants.BuildActionCode(permissionconstants.ResourceTypeForm, permissionconstants.ActionRead):  "表单查看",
		permissionconstants.BuildActionCode(permissionconstants.ResourceTypeForm, permissionconstants.ActionWrite): "表单提交",
		permissionconstants.BuildActionCode(permissionconstants.ResourceTypeForm, permissionconstants.ActionAdmin): "表单管理",
		// Chart 函数操作
		permissionconstants.BuildActionCode(permissionconstants.ResourceTypeChart, permissionconstants.ActionRead):  "图表查看",
		permissionconstants.BuildActionCode(permissionconstants.ResourceTypeChart, permissionconstants.ActionAdmin): "图表管理",
		// Directory 操作
		permissionconstants.BuildActionCode(permissionconstants.ResourceTypeDirectory, permissionconstants.ActionRead):   "目录查看",
		permissionconstants.BuildActionCode(permissionconstants.ResourceTypeDirectory, permissionconstants.ActionWrite):  "目录写入",
		permissionconstants.BuildActionCode(permissionconstants.ResourceTypeDirectory, permissionconstants.ActionUpdate): "目录更新",
		permissionconstants.BuildActionCode(permissionconstants.ResourceTypeDirectory, permissionconstants.ActionDelete): "目录删除",
		permissionconstants.BuildActionCode(permissionconstants.ResourceTypeDirectory, permissionconstants.ActionAdmin):  "目录管理",
		// App 操作
		permissionconstants.BuildActionCode(permissionconstants.ResourceTypeApp, permissionconstants.ActionRead):   "工作空间查看",
		permissionconstants.BuildActionCode(permissionconstants.ResourceTypeApp, permissionconstants.ActionWrite):  "工作空间创建",
		permissionconstants.BuildActionCode(permissionconstants.ResourceTypeApp, permissionconstants.ActionUpdate): "工作空间更新",
		permissionconstants.BuildActionCode(permissionconstants.ResourceTypeApp, permissionconstants.ActionDelete): "工作空间删除",
		permissionconstants.BuildActionCode(permissionconstants.ResourceTypeApp, permissionconstants.ActionAdmin):  "工作空间管理",
		// Docs 操作
		permissionconstants.BuildActionCode(permissionconstants.ResourceTypeDocs, permissionconstants.ActionRead):   "文档查看",
		permissionconstants.BuildActionCode(permissionconstants.ResourceTypeDocs, permissionconstants.ActionWrite):  "文档编辑",
		permissionconstants.BuildActionCode(permissionconstants.ResourceTypeDocs, permissionconstants.ActionDelete): "文档删除",
		permissionconstants.BuildActionCode(permissionconstants.ResourceTypeDocs, permissionconstants.ActionAdmin):  "文档管理",
		// Board 讨论区操作
		permissionconstants.BuildActionCode(permissionconstants.ResourceTypeBoard, permissionconstants.ActionRead):   "帖子查看",
		permissionconstants.BuildActionCode(permissionconstants.ResourceTypeBoard, permissionconstants.ActionWrite):  "发帖",
		permissionconstants.BuildActionCode(permissionconstants.ResourceTypeBoard, permissionconstants.ActionUpdate): "帖子更新",
		permissionconstants.BuildActionCode(permissionconstants.ResourceTypeBoard, permissionconstants.ActionDelete): "帖子删除",
		permissionconstants.BuildActionCode(permissionconstants.ResourceTypeBoard, permissionconstants.ActionAdmin):  "板块管理",
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

// ==================== 文档权限中间件 ====================

// CheckDocRead 检查文档读取权限
// 权限点：docs:read
func CheckDocRead() gin.HandlerFunc {
	return func(c *gin.Context) {
		if skipPermissionChecks(c) {
			return
		}
		action := permissionconstants.BuildActionCode(permissionconstants.ResourceTypeDocs, permissionconstants.ActionRead)
		if !checkPermission(c, action, "无权限查看该文档") {
			return
		}
		c.Next()
	}
}

// CheckDocWrite 检查文档写入权限
// 权限点：docs:write
func CheckDocWrite() gin.HandlerFunc {
	return func(c *gin.Context) {
		if skipPermissionChecks(c) {
			return
		}
		action := permissionconstants.BuildActionCode(permissionconstants.ResourceTypeDocs, permissionconstants.ActionWrite)
		if !checkPermission(c, action, "无权限编辑该文档") {
			return
		}
		c.Next()
	}
}

// CheckDocDelete 检查文档删除权限
// 权限点：docs:delete
func CheckDocDelete() gin.HandlerFunc {
	return func(c *gin.Context) {
		if skipPermissionChecks(c) {
			return
		}
		action := permissionconstants.BuildActionCode(permissionconstants.ResourceTypeDocs, permissionconstants.ActionDelete)
		if !checkPermission(c, action, "无权限删除该文档") {
			return
		}
		c.Next()
	}
}

// ==================== 讨论区/板块权限中间件 ====================

// CheckBoardRead 检查讨论区查看权限（从 query full_code_path 取路径）
// 权限点：board:read
func CheckBoardRead() gin.HandlerFunc {
	return func(c *gin.Context) {
		if skipPermissionChecks(c) {
			return
		}
		fullCodePath := c.Query("full_code_path")
		if fullCodePath == "" {
			response.PermissionDenied(c, "缺少版块路径参数 full_code_path", map[string]interface{}{
				"resource_path": "",
				"action":        "board:read",
			})
			return
		}
		action := permissionconstants.BuildActionCode(permissionconstants.ResourceTypeBoard, "read")
		if !checkPermissionWithPath(c, fullCodePath, action, "无权限查看该讨论区") {
			return
		}
		c.Next()
	}
}

// CheckBoardWrite 检查讨论区发帖权限（从 body full_code_path 取路径，不消费 body）
// 权限点：board:write
func CheckBoardWrite() gin.HandlerFunc {
	return func(c *gin.Context) {
		if skipPermissionChecks(c) {
			return
		}
		if c.Request.Body == nil {
			response.PermissionDenied(c, "请求体为空", map[string]interface{}{"action": "board:write"})
			return
		}
		// 读取并恢复 body，只解析 full_code_path
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			response.PermissionDenied(c, "读取请求体失败", map[string]interface{}{"action": "board:write"})
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
		var req struct {
			FullCodePath string `json:"full_code_path"`
		}
		if err := json.Unmarshal(body, &req); err != nil || req.FullCodePath == "" {
			response.PermissionDenied(c, "缺少版块路径 full_code_path", map[string]interface{}{"action": "board:write"})
			return
		}
		action := permissionconstants.BuildActionCode(permissionconstants.ResourceTypeBoard, "write")
		if !checkPermissionWithPath(c, req.FullCodePath, action, "无权限在该讨论区发帖") {
			return
		}
		c.Next()
	}
}

// GetPathByPostID 根据帖子 ID 解析版块路径（由 router 注入，依赖 BoardService）
type GetPathByPostID func(c *gin.Context, id int64) (string, error)

// CheckBoardReadFromPostID 检查讨论区查看权限（根据帖子 id 解析版块路径）
// 权限点：board:read
func CheckBoardReadFromPostID(getPath GetPathByPostID) gin.HandlerFunc {
	return checkBoardPermissionFromPostID(getPath, "read", "无权限查看该帖子")
}

// CheckBoardUpdateFromPostID 检查讨论区更新权限（根据帖子 id 解析版块路径）
// 权限点：board:update
func CheckBoardUpdateFromPostID(getPath GetPathByPostID) gin.HandlerFunc {
	return checkBoardPermissionFromPostID(getPath, "update", "无权限更新该帖子")
}

// CheckBoardDeleteFromPostID 检查讨论区删除权限（根据帖子 id 解析版块路径）
// 权限点：board:delete
func CheckBoardDeleteFromPostID(getPath GetPathByPostID) gin.HandlerFunc {
	return checkBoardPermissionFromPostID(getPath, "delete", "无权限删除该帖子")
}

func checkBoardPermissionFromPostID(getPath GetPathByPostID, actionType string, errorMsg string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if skipPermissionChecks(c) {
			return
		}
		idStr := c.Param("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || id <= 0 {
			response.PermissionDenied(c, "无效的帖子ID", map[string]interface{}{"action": "board:" + actionType})
			return
		}
		fullCodePath, err := getPath(c, id)
		if err != nil {
			response.FailWithMessage(c, err.Error())
			return
		}
		if fullCodePath == "" {
			response.PermissionDenied(c, "无法解析版块路径", map[string]interface{}{"action": "board:" + actionType})
			return
		}
		action := permissionconstants.BuildActionCode(permissionconstants.ResourceTypeBoard, actionType)
		if !checkPermissionWithPath(c, fullCodePath, action, errorMsg) {
			return
		}
		c.Next()
	}
}
