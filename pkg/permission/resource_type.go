package permission

import "github.com/ai-agent-os/ai-agent-os/pkg/servicetree"

// 资源类型常量
const (
	ResourceTypeDirectory = "directory" // 目录
	ResourceTypeTable     = "table"     // 表格函数
	ResourceTypeForm      = "form"      // 表单函数
	ResourceTypeChart     = "chart"     // 图表函数
	ResourceTypeDocs      = "docs"      // 文档
	ResourceTypeBoard     = "board"     // 讨论区/板块
	ResourceTypeApp       = "app"       // 工作空间
)

// GetResourceType 根据节点类型和模板类型获取资源类型
func GetResourceType(nodeType string, templateType string) string {
	if nodeType == servicetree.TypePackage || nodeType == ResourceTypeDirectory {
		return ResourceTypeDirectory
	} else if nodeType == servicetree.TypeFunction {
		switch templateType {
		case ResourceTypeTable:
			return ResourceTypeTable
		case ResourceTypeForm:
			return ResourceTypeForm
		case ResourceTypeChart:
			return ResourceTypeChart
		default:
			return ResourceTypeTable // 默认使用 table
		}
	} else if nodeType == servicetree.TypeDocs {
		return ResourceTypeDocs
	} else if nodeType == servicetree.TypeBoard {
		return ResourceTypeBoard
	} else if nodeType == ResourceTypeApp {
		return ResourceTypeApp
	}
	return ""
}

// GetActionsForResourceType 根据资源类型获取可用的权限点列表
// ⭐ 权限点格式：resource_type:action_type（如 form:read, table:write）
func GetActionsForResourceType(resourceType string) []string {
	switch resourceType {
	case ResourceTypeDirectory:
		return BuildActionCodes(ResourceTypeDirectory, ActionRead, ActionWrite, ActionUpdate, ActionDelete, ActionAdmin)
	case ResourceTypeTable:
		return BuildActionCodes(ResourceTypeTable, ActionRead, ActionWrite, ActionUpdate, ActionDelete, ActionAdmin)
	case ResourceTypeForm:
		// Form 函数只支持 read、write、admin
		return BuildActionCodes(ResourceTypeForm, ActionRead, ActionWrite, ActionAdmin)
	case ResourceTypeChart:
		// Chart 函数只支持 read、admin
		return BuildActionCodes(ResourceTypeChart, ActionRead, ActionAdmin)
	case ResourceTypeDocs:
		// Docs 文档支持 read、write、delete、admin
		return BuildActionCodes(ResourceTypeDocs, ActionRead, ActionWrite, ActionDelete, ActionAdmin)
	case ResourceTypeBoard:
		// Board 讨论区支持 read、write、update、delete、admin
		return BuildActionCodes(ResourceTypeBoard, ActionRead, ActionWrite, ActionUpdate, ActionDelete, ActionAdmin)
	case ResourceTypeApp:
		return BuildActionCodes(ResourceTypeApp, ActionRead, ActionWrite, ActionUpdate, ActionDelete, ActionAdmin)
	default:
		return []string{}
	}
}

// BuildActionCode 构建权限点编码（resource_type:action_type）
func BuildActionCode(resourceType string, actionType string) string {
	return resourceType + ":" + actionType
}

// BuildActionCodes 批量构建权限点编码。
func BuildActionCodes(resourceType string, actionTypes ...string) []string {
	result := make([]string, 0, len(actionTypes))
	for _, actionType := range actionTypes {
		result = append(result, BuildActionCode(resourceType, actionType))
	}
	return result
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
		ResourceTypeDocs,
		ResourceTypeBoard,
		ResourceTypeApp,
	}
}
