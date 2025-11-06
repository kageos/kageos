package serviceconfig

import (
	"os"
	"strings"
)

// GetGatewayURL 获取网关地址
// 优先级：环境变量 > 默认值
func GetGatewayURL() string {
	// 优先环境变量（生产环境）
	if url := os.Getenv("GATEWAY_URL"); url != "" {
		return url
	}

	// 默认值（开发环境）
	return "http://localhost:9090"
}

// GetServiceURL 获取服务地址（通过网关）
// serviceName: 服务名称（如 "storage", "app"）
// path: 服务路径前缀（如 "/api/v1/storage", "/api"）
func GetServiceURL(serviceName, path string) string {
	gatewayURL := GetGatewayURL()

	// 确保路径以 / 开头
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// 返回：网关地址 + 路径
	return gatewayURL + path
}

// GetStorageURL 获取存储服务地址（便捷方法）
func GetStorageURL() string {
	return GetServiceURL("storage", "/api/v1/storage")
}

// GetAppServerURL 获取主服务地址（便捷方法）
func GetAppServerURL() string {
	return GetServiceURL("app", "/api")
}

// BuildServiceURL 构建完整的服务 URL
// gatewayURL: 网关地址
// path: 服务路径（如 "/api/v1/storage/upload"）
func BuildServiceURL(gatewayURL, path string) string {
	// 确保路径以 / 开头
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// 移除网关地址末尾的 /
	gatewayURL = strings.TrimSuffix(gatewayURL, "/")

	return gatewayURL + path
}
