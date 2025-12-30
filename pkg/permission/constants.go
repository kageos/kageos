package permission

// ⭐ 权限常量定义
// 集中管理所有权限点，避免硬编码和拼写错误

// Function 权限（统一权限点：所有函数类型统一使用）
const (
	FunctionRead   = "function:read"   // 函数查看
	FunctionWrite  = "function:write"  // 函数写入（table 新增记录、form 提交）
	FunctionUpdate = "function:update" // 函数更新（table 更新记录）
	FunctionDelete = "function:delete" // 函数删除（table 删除记录）
	FunctionManage = "function:manage" // 函数所有权（拥有所有操作权限）
)

// Directory 权限
const (
	DirectoryRead   = "directory:read"   // 目录查看
	DirectoryWrite  = "directory:write"  // 目录写入（创建子目录和函数）
	DirectoryUpdate = "directory:update" // 目录更新（修改目录信息）
	DirectoryDelete = "directory:delete" // 目录删除
	DirectoryManage = "directory:manage" // 目录所有权（拥有所有目录权限）
)

// App 权限（工作空间权限）
const (
	AppRead   = "app:read"   // 工作空间查看
	AppCreate = "app:create" // 工作空间创建
	AppUpdate = "app:update" // 工作空间更新
	AppDelete = "app:delete" // 工作空间删除
	AppDeploy = "app:deploy" // 工作空间部署
	AppManage = "app:manage" // 工作空间所有权（拥有所有工作空间权限）
)

// 权限点列表（用于批量查询）
var (
	// FunctionActions 函数权限点列表
	FunctionActions = []string{
		FunctionRead,
		FunctionWrite,
		FunctionUpdate,
		FunctionDelete,
		FunctionManage,
	}

	// DirectoryActions 目录权限点列表
	DirectoryActions = []string{
		DirectoryRead,
		DirectoryWrite,
		DirectoryUpdate,
		DirectoryDelete,
		DirectoryManage,
	}

	// AppActions 工作空间权限点列表
	AppActions = []string{
		AppRead,
		AppCreate,
		AppUpdate,
		AppDelete,
		AppDeploy,
		AppManage,
	}

	// AllActions 所有权限点列表（用于查询所有权限）
	AllActions = []string{
		// Function 权限
		FunctionRead,
		FunctionWrite,
		FunctionUpdate,
		FunctionDelete,
		FunctionManage,
		// Directory 权限
		DirectoryRead,
		DirectoryWrite,
		DirectoryUpdate,
		DirectoryDelete,
		DirectoryManage,
		// App 权限
		AppRead,
		AppCreate,
		AppUpdate,
		AppDelete,
		AppDeploy,
		AppManage,
	}
)

