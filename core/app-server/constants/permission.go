package constants

/**
 * 权限常量定义
 *
 * 说明：
 * 1. 所有权限点格式：resource_type:action_type
 * 2. 资源类型：directory、function、table、form、chart、docs
 * 3. 操作类型：read、write、update、delete、admin、query
 *
 * ⭐ 前后端统一使用此常量定义，避免硬编码
 * ⭐ 对应前端文件：web/src/constants/permissions.ts
 */

// ========================================
// 资源类型常量
// ========================================

const (
	// ResourceTypeDirectory 目录类型（包括根目录/工作空间）
	ResourceTypeDirectory = "directory"
	// ResourceTypeFunction 函数类型
	ResourceTypeFunction = "function"
	// ResourceTypeTable 表格类型
	ResourceTypeTable = "table"
	// ResourceTypeForm 表单类型
	ResourceTypeForm = "form"
	// ResourceTypeChart 图表类型
	ResourceTypeChart = "chart"
	// ResourceTypeDocs 文档类型
	ResourceTypeDocs = "docs"
	// ResourceTypeBoard 讨论区/板块类型
	ResourceTypeBoard = "board"
)

// ========================================
// 操作类型常量
// ========================================

const (
	// ActionTypeRead 查看权限
	ActionTypeRead = "read"
	// ActionTypeWrite 写入权限
	ActionTypeWrite = "write"
	// ActionTypeUpdate 更新权限
	ActionTypeUpdate = "update"
	// ActionTypeDelete 删除权限
	ActionTypeDelete = "delete"
	// ActionTypeAdmin 管理员权限（包含所有权限）
	ActionTypeAdmin = "admin"
	// ActionTypeQuery 查询权限（用于 chart）
	ActionTypeQuery = "query"
)

// ========================================
// 目录权限常量
// ========================================

const (
	// PermissionDirectoryRead 查看目录
	PermissionDirectoryRead = "directory:read"
	// PermissionDirectoryWrite 创建子目录/子资源
	PermissionDirectoryWrite = "directory:write"
	// PermissionDirectoryUpdate 更新目录信息
	PermissionDirectoryUpdate = "directory:update"
	// PermissionDirectoryDelete 删除目录
	PermissionDirectoryDelete = "directory:delete"
	// PermissionDirectoryAdmin 目录管理员（拥有所有权限）
	PermissionDirectoryAdmin = "directory:admin"
)

// ========================================
// 函数权限常量
// ========================================

const (
	// PermissionFunctionRead 查看函数
	PermissionFunctionRead = "function:read"
	// PermissionFunctionWrite 创建/写入函数
	PermissionFunctionWrite = "function:write"
	// PermissionFunctionUpdate 更新函数
	PermissionFunctionUpdate = "function:update"
	// PermissionFunctionDelete 删除函数
	PermissionFunctionDelete = "function:delete"
	// PermissionFunctionAdmin 函数管理员（拥有所有权限）
	PermissionFunctionAdmin = "function:admin"
)

// ========================================
// 表格权限常量
// ========================================

const (
	// PermissionTableRead 查看表格
	PermissionTableRead = "table:read"
	// PermissionTableWrite 插入数据
	PermissionTableWrite = "table:write"
	// PermissionTableUpdate 更新数据
	PermissionTableUpdate = "table:update"
	// PermissionTableDelete 删除数据
	PermissionTableDelete = "table:delete"
	// PermissionTableAdmin 表格管理员（拥有所有权限）
	PermissionTableAdmin = "table:admin"
)

// ========================================
// 表单权限常量
// ========================================

const (
	// PermissionFormRead 查看表单
	PermissionFormRead = "form:read"
	// PermissionFormWrite 提交表单
	PermissionFormWrite = "form:write"
	// PermissionFormUpdate 更新表单数据
	PermissionFormUpdate = "form:update"
	// PermissionFormDelete 删除表单数据
	PermissionFormDelete = "form:delete"
	// PermissionFormAdmin 表单管理员（拥有所有权限）
	PermissionFormAdmin = "form:admin"
)

// ========================================
// 图表权限常量
// ========================================

const (
	// PermissionChartRead 查看图表
	PermissionChartRead = "chart:read"
	// PermissionChartQuery 查询图表数据
	PermissionChartQuery = "chart:query"
	// PermissionChartUpdate 更新图表配置
	PermissionChartUpdate = "chart:update"
	// PermissionChartDelete 删除图表
	PermissionChartDelete = "chart:delete"
	// PermissionChartAdmin 图表管理员（拥有所有权限）
	PermissionChartAdmin = "chart:admin"
)

// ========================================
// 文档权限常量
// ========================================

const (
	// PermissionDocsRead 查看文档
	PermissionDocsRead = "docs:read"
	// PermissionDocsWrite 创建/写入文档
	PermissionDocsWrite = "docs:write"
	// PermissionDocsUpdate 更新文档
	PermissionDocsUpdate = "docs:update"
	// PermissionDocsDelete 删除文档
	PermissionDocsDelete = "docs:delete"
	// PermissionDocsAdmin 文档管理员（拥有所有权限）
	PermissionDocsAdmin = "docs:admin"
)

// ========================================
// 讨论区/板块权限常量
// ========================================

const (
	// PermissionBoardRead 查看帖子
	PermissionBoardRead = "board:read"
	// PermissionBoardWrite 发帖
	PermissionBoardWrite = "board:write"
	// PermissionBoardUpdate 更新帖子
	PermissionBoardUpdate = "board:update"
	// PermissionBoardDelete 删除帖子
	PermissionBoardDelete = "board:delete"
	// PermissionBoardAdmin 板块管理员（拥有所有权限）
	PermissionBoardAdmin = "board:admin"
)

// ========================================
// 工具函数
// ========================================

// BuildPermission 构建权限点
// resourceType: 资源类型（如 "directory"）
// actionType: 操作类型（如 "read"）
// 返回: 权限点字符串（如 "directory:read"）
func BuildPermission(resourceType, actionType string) string {
	return resourceType + ":" + actionType
}

// ParsePermission 解析权限点
// permission: 权限点字符串（如 "directory:read"）
// 返回: resourceType, actionType, ok
func ParsePermission(permission string) (resourceType, actionType string, ok bool) {
	for i := 0; i < len(permission); i++ {
		if permission[i] == ':' {
			resourceType = permission[:i]
			actionType = permission[i+1:]
			ok = true
			return
		}
	}
	return "", "", false
}

// GetAllResourceTypes 获取所有资源类型
func GetAllResourceTypes() []string {
	return []string{
		ResourceTypeDirectory,
		ResourceTypeFunction,
		ResourceTypeTable,
		ResourceTypeForm,
		ResourceTypeChart,
		ResourceTypeDocs,
		ResourceTypeBoard,
	}
}

// GetAllActionTypes 获取所有操作类型
func GetAllActionTypes() []string {
	return []string{
		ActionTypeRead,
		ActionTypeWrite,
		ActionTypeUpdate,
		ActionTypeDelete,
		ActionTypeAdmin,
		ActionTypeQuery,
	}
}

// GetPermissionsByResourceType 根据资源类型获取所有权限点
func GetPermissionsByResourceType(resourceType string) []string {
	switch resourceType {
	case ResourceTypeDirectory:
		return []string{
			PermissionDirectoryRead,
			PermissionDirectoryWrite,
			PermissionDirectoryUpdate,
			PermissionDirectoryDelete,
			PermissionDirectoryAdmin,
		}
	case ResourceTypeFunction:
		return []string{
			PermissionFunctionRead,
			PermissionFunctionWrite,
			PermissionFunctionUpdate,
			PermissionFunctionDelete,
			PermissionFunctionAdmin,
		}
	case ResourceTypeTable:
		return []string{
			PermissionTableRead,
			PermissionTableWrite,
			PermissionTableUpdate,
			PermissionTableDelete,
			PermissionTableAdmin,
		}
	case ResourceTypeForm:
		return []string{
			PermissionFormRead,
			PermissionFormWrite,
			PermissionFormUpdate,
			PermissionFormDelete,
			PermissionFormAdmin,
		}
	case ResourceTypeChart:
		return []string{
			PermissionChartRead,
			PermissionChartQuery,
			PermissionChartUpdate,
			PermissionChartDelete,
			PermissionChartAdmin,
		}
	case ResourceTypeDocs:
		return []string{
			PermissionDocsRead,
			PermissionDocsWrite,
			PermissionDocsUpdate,
			PermissionDocsDelete,
			PermissionDocsAdmin,
		}
	case ResourceTypeBoard:
		return []string{
			PermissionBoardRead,
			PermissionBoardWrite,
			PermissionBoardUpdate,
			PermissionBoardDelete,
			PermissionBoardAdmin,
		}
	default:
		return []string{}
	}
}
