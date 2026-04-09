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
	return config.GetGatewayURL()
}

// BuildGatewayURL 基于当前网关配置构建完整 URL。
func BuildGatewayURL(path string) string {
	return joinURL(GetGatewayURL(), path)
}

func joinURL(baseURL, path string) string {
	// 确保路径以 / 开头
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	// 移除基地址末尾的 /
	baseURL = strings.TrimSuffix(baseURL, "/")

	return baseURL + path
}
