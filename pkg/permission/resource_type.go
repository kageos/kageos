package permission

// 资源类型常量
const (
	ResourceTypeDirectory = "directory" // 目录
	ResourceTypeTable     = "table"     // 表格函数
	ResourceTypeForm      = "form"      // 表单函数
	ResourceTypeChart     = "chart"     // 图表函数
	ResourceTypeApp       = "app"       // 工作空间
)

// GetResourceType 根据节点类型和模板类型获取资源类型
func GetResourceType(nodeType string, templateType string) string {
	if nodeType == "package" || nodeType == "directory" {
		return ResourceTypeDirectory
	} else if nodeType == "function" {
		switch templateType {
		case "table":
			return ResourceTypeTable
		case "form":
			return ResourceTypeForm
		case "chart":
			return ResourceTypeChart
		default:
			return ResourceTypeTable // 默认使用 table
		}
	} else if nodeType == "app" {
		return ResourceTypeApp
	}
	return ""
}

// GetActionsForResourceType 根据资源类型获取可用的权限点列表
// ⭐ 权限点格式：resource_type:action_type（如 form:read, table:write）
func GetActionsForResourceType(resourceType string) []string {
	switch resourceType {
	case ResourceTypeDirectory:
		return []string{
			BuildActionCode(ResourceTypeDirectory, "read"),
			BuildActionCode(ResourceTypeDirectory, "write"),
			BuildActionCode(ResourceTypeDirectory, "update"),
			BuildActionCode(ResourceTypeDirectory, "delete"),
			BuildActionCode(ResourceTypeDirectory, "admin"),
		}
	case ResourceTypeTable:
		return []string{
			BuildActionCode(ResourceTypeTable, "read"),
			BuildActionCode(ResourceTypeTable, "write"),
			BuildActionCode(ResourceTypeTable, "update"),
			BuildActionCode(ResourceTypeTable, "delete"),
			BuildActionCode(ResourceTypeTable, "admin"),
		}
	case ResourceTypeForm:
		// Form 函数只支持 read、write、admin
		return []string{
			BuildActionCode(ResourceTypeForm, "read"),
			BuildActionCode(ResourceTypeForm, "write"),
			BuildActionCode(ResourceTypeForm, "admin"),
		}
	case ResourceTypeChart:
		// Chart 函数只支持 read、admin
		return []string{
			BuildActionCode(ResourceTypeChart, "read"),
			BuildActionCode(ResourceTypeChart, "admin"),
		}
	case ResourceTypeApp:
		return []string{
			BuildActionCode(ResourceTypeApp, "read"),
			BuildActionCode(ResourceTypeApp, "write"),
			BuildActionCode(ResourceTypeApp, "update"),
			BuildActionCode(ResourceTypeApp, "delete"),
			BuildActionCode(ResourceTypeApp, "admin"),
		}
	default:
		return []string{}
	}
}

// BuildActionCode 构建权限点编码（resource_type:action_type）
func BuildActionCode(resourceType string, actionType string) string {
	return resourceType + ":" + actionType
}

// ParseActionCode 解析权限点编码，返回资源类型和操作类型
func ParseActionCode(code string) (resourceType string, actionType string, ok bool) {
	// 格式：resource_type:action_type
	// 例如：form:read, table:write
	for i := 0; i < len(code); i++ {
		if code[i] == ':' {
			resourceType = code[:i]
			actionType = code[i+1:]
			ok = true
			return
		}
	}
	return "", "", false
}

// GetAllResourceTypes 获取所有资源类型列表
func GetAllResourceTypes() []string {
	return []string{
		ResourceTypeDirectory,
		ResourceTypeTable,
		ResourceTypeForm,
		ResourceTypeChart,
		ResourceTypeApp,
	}
}
