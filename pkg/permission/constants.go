package permission

// ⭐ 权限常量定义（简化版）
// 权限点命名已简化：read, write, update, delete, admin
// 因为角色已按资源类型分组，不需要在权限点中重复资源类型信息

// 通用权限点（所有资源类型统一使用）
const (
	ActionRead   = "read"   // 读权限
	ActionWrite  = "write"  // 写权限（创建、提交）
	ActionUpdate = "update" // 更新权限（修改）
	ActionDelete = "delete" // 删除权限
	ActionAdmin  = "admin" // 管理员权限（拥有所有权限）
)


// 权限点列表（用于批量查询）
var (
	// FunctionActions 函数权限点列表（简化版）
	FunctionActions = []string{
		ActionRead,
		ActionWrite,
		ActionUpdate,
		ActionDelete,
		ActionAdmin,
	}

	// DirectoryActions 目录权限点列表（简化版）
	DirectoryActions = []string{
		ActionRead,
		ActionWrite,
		ActionUpdate,
		ActionDelete,
		ActionAdmin,
	}

	// AppActions 工作空间权限点列表（简化版）
	AppActions = []string{
		ActionRead,
		ActionWrite,  // create 使用 write
		ActionUpdate,
		ActionDelete,
		ActionAdmin,
	}

	// AllActions 所有权限点列表（用于查询所有权限）
	AllActions = []string{
		ActionRead,
		ActionWrite,
		ActionUpdate,
		ActionDelete,
		ActionAdmin,
	}
)

