package permission

// GetActionsForNode 根据节点类型获取权限点列表
// 统一实现，避免代码重复
//
// 参数：
//   - nodeType: 节点类型（"package" 或 "function"）
//   - templateType: 模板类型（"table"、"form"、"chart"，仅对 function 有效）
//
// 返回：
//   - []string: 需要检查的权限点列表
//
// 说明：
//   - 服务目录（package）：返回 directory 相关权限
//   - 函数（function）：返回 function 相关权限（统一权限点，不区分模板类型）
func GetActionsForNode(nodeType string, templateType string) []string {
	if nodeType == "package" {
		// 服务目录（package）：检查目录权限
		return DirectoryActions
	} else if nodeType == "function" {
		// ⭐ 统一权限点：所有函数类型统一使用 function:read/write/update/delete
		// 不区分 templateType，统一返回所有 function 权限点
		return FunctionActions
	}

	// 未知类型，返回空列表
	return []string{}
}

