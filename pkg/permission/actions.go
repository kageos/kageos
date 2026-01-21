package permission

// GetActionsForNode 根据节点类型获取权限点列表
// ⭐ 权限点格式：resource_type:action_type（如 form:read, table:write）
//
// 参数：
//   - nodeType: 节点类型（"package" 或 "function"）
//   - templateType: 模板类型（"table"、"form"、"chart"，仅对 function 有效）
//
// 返回：
//   - []string: 需要检查的权限点列表（格式：resource_type:action_type）
func GetActionsForNode(nodeType string, templateType string) []string {
	// 根据节点类型和模板类型获取资源类型
	resourceType := GetResourceType(nodeType, templateType)
	if resourceType == "" {
		return []string{}
	}
	
	// 返回该资源类型的权限点列表
	return GetActionsForResourceType(resourceType)
}

