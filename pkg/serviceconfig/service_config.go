package serviceconfig

import (
	"os"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/pkg/config"
)

// GetGatewayURL 获取网关地址
// 优先级：环境变量 > 全局配置 > 默认值
// 注意：服务运行在裸机上，使用 127.0.0.1 访问
func GetGatewayURL() string {
	// 优先环境变量（生产环境）
	if url := os.Getenv("GATEWAY_URL"); url != "" {
		return url
	}

	// 从全局配置读取（裸机服务访问）
	globalConfig := config.GetGlobalSharedConfig()
	gatewayURL := globalConfig.Gateway.GetBaseURL()
	if gatewayURL != "" {
		return gatewayURL
	}

	// 默认值（开发环境）
	return "http://127.0.0.1:9090"
}

// GetServiceURL 获取服务地址（通过网关）
// serviceName: 服务名称（如 "storage", "app"）
// path: 服务路径前缀（如 "/storage/api/v1", "/api"）
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
	return GetServiceURL("storage", "/storage/api/v1")
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
