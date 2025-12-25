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
//   - isFunction: true (路径深度 >= 4)
func ParseFullCodePath(fullCodePath string) (pathParts []string, user string, app string, isFunction bool) {
	pathParts = strings.Split(strings.Trim(fullCodePath, "/"), "/")
	
	if len(pathParts) >= 2 {
		user = pathParts[0]
		app = pathParts[1]
	}
	
	// 路径深度 >= 4 时，可能是函数（/user/app/dir/function）
	isFunction = len(pathParts) >= 4
	
	return pathParts, user, app, isFunction
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

// GetDirectoryPath 获取目录路径（如果是函数，返回父目录；如果是目录，返回自身）
func GetDirectoryPath(fullCodePath string) string {
	pathParts := strings.Split(strings.Trim(fullCodePath, "/"), "/")
	if len(pathParts) >= 3 {
		// 如果是函数（>= 4），返回父目录；如果是目录（= 3），返回自身
		if len(pathParts) >= 4 {
			// 函数，返回父目录
			return "/" + strings.Join(pathParts[:len(pathParts)-1], "/")
		}
		// 目录，返回自身
		return "/" + strings.Join(pathParts, "/")
	}
	return ""
}

