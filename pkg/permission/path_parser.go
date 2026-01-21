package permission

import (
	"strings"
)

// ParseFullCodePath 解析 full-code-path，提取路径信息
// 例如：/luobei/operations/procurement/order/cashier_product_list
// 返回：
//   - pathParts: ["luobei", "operations", "procurement", "order", "cashier_product_list"]
//   - user: "luobei"
//   - app: "operations"
// ⭐ 注意：不返回 isFunction，因为无法通过路径深度判断资源类型，需要通过 ServiceTree 查询确定
func ParseFullCodePath(fullCodePath string) (pathParts []string, user string, app string) {
	pathParts = strings.Split(strings.Trim(fullCodePath, "/"), "/")
	
	if len(pathParts) >= 2 {
		user = pathParts[0]
		app = pathParts[1]
	}
	
	return pathParts, user, app
}

// GetParentPaths 获取所有父目录路径（从直接父目录到应用级别）
// 例如：/luobei/operations/procurement/order/cashier_product_list
// 返回：
//   - ["/luobei/operations/procurement/order", "/luobei/operations/procurement", "/luobei/operations"]
func GetParentPaths(fullCodePath string) []string {
	pathParts := strings.Split(strings.Trim(fullCodePath, "/"), "/")
	
	if len(pathParts) < 4 {
		// 路径深度 < 4，没有父目录（或父目录是应用级别）
		return []string{}
	}
	
	parentPaths := make([]string, 0)
	// 从直接父目录开始，向上查找所有父目录（至少包含 user/app）
	for i := len(pathParts) - 1; i >= 2; i-- {
		parentPath := "/" + strings.Join(pathParts[:i], "/")
		parentPaths = append(parentPaths, parentPath)
	}
	
	return parentPaths
}

// GetAppPath 获取应用路径（/user/app）
func GetAppPath(fullCodePath string) string {
	pathParts := strings.Split(strings.Trim(fullCodePath, "/"), "/")
	if len(pathParts) >= 2 {
		return "/" + strings.Join(pathParts[:2], "/")
	}
	return ""
}

// GetDirectoryPath 获取目录路径
// ⭐ 注意：此函数假设路径深度 >= 4 时最后一段是函数名，返回父目录；否则返回自身
// 实际使用时应该通过 ServiceTree 查询确定资源类型，而不是通过路径深度猜测
func GetDirectoryPath(fullCodePath string) string {
	pathParts := strings.Split(strings.Trim(fullCodePath, "/"), "/")
	if len(pathParts) >= 3 {
		// ⭐ 警告：这里通过路径深度猜测，不准确！应该通过 ServiceTree 查询确定
		// 如果路径深度 >= 4，假设最后一段是函数名，返回父目录
		if len(pathParts) >= 4 {
			return "/" + strings.Join(pathParts[:len(pathParts)-1], "/")
		}
		// 否则返回自身
		return "/" + strings.Join(pathParts, "/")
	}
	return ""
}

