package serviceconfig

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/netprobe"
)

// GetGatewayURL 获取网关地址
// 优先级：环境变量 > 全局配置 > 默认值
// 本地地址会自动探测 127.0.0.1 / host.containers.internal 等候选，
// 避免 SDK app 在 host/bridge 网络之间切换时使用不可达地址。
func GetGatewayURL() string {
	// 优先环境变量（生产环境）
	if url := os.Getenv("GATEWAY_URL"); url != "" {
		return resolveGatewayURL(url)
	}
	return resolveGatewayURL(config.GetGatewayURL())
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

func resolveGatewayURL(baseURL string) string {
	if len(netprobe.URLCandidates(baseURL)) <= 1 {
		return baseURL
	}
	resolved, err := netprobe.ResolveHTTPBaseURLCached(context.Background(), "gateway", baseURL, "/health", time.Second)
	if err != nil {
		return baseURL
	}
	return resolved
}
