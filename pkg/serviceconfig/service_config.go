package serviceconfig

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/netprobe"
)

// GetGatewayURL 获取网关地址。
//
// 核心服务使用全局配置中的权威地址。SDK app 容器会显式注入
// GATEWAY_URL；只有这种运行时地址才需要在 127.0.0.1、
// host.containers.internal 等候选之间自动探测。
func GetGatewayURL() string {
	if runtimeURL, ok := gatewayRuntimeURL(); ok {
		return resolveGatewayURL(runtimeURL)
	}
	return config.GetGatewayURL()
}

// InvalidateGatewayURL makes the next gateway request probe local runtime
// candidates again. The failed request itself is never replayed here.
func InvalidateGatewayURL() {
	baseURL, ok := gatewayRuntimeURL()
	if !ok {
		return
	}
	if len(netprobe.URLCandidates(baseURL)) > 1 {
		netprobe.InvalidateHTTPBaseURLCached("gateway", baseURL, "/health")
	}
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

func gatewayRuntimeURL() (string, bool) {
	if configured := strings.TrimSpace(os.Getenv("GATEWAY_URL")); configured != "" {
		return configured, true
	}
	return "", false
}
