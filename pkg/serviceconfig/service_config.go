package serviceconfig

import (
	"context"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/netprobe"
)

// GetGatewayURL 获取网关地址
// 优先级：环境变量 > 全局配置 > 默认值
// 本地地址会自动探测 127.0.0.1 / host.containers.internal 等候选，
// 避免 SDK app 在 host/bridge 网络之间切换时使用不可达地址。
func GetGatewayURL() string {
	return resolveGatewayURL(gatewayBaseURL())
}

// InvalidateGatewayURL makes the next gateway request probe local runtime
// candidates again. The failed request itself is never replayed here.
func InvalidateGatewayURL() {
	baseURL := gatewayBaseURL()
	if len(netprobe.URLCandidates(baseURL)) > 1 {
		netprobe.InvalidateHTTPBaseURLCached("gateway", baseURL, "/health")
	}
}

// BuildGatewayURL 基于当前网关配置构建完整 URL。
func BuildGatewayURL(path string) string {
	return joinURL(GetGatewayURL(), path)
}

// GetInternalTimerSchedulerURL returns the timer scheduler's local service
// address. Core services use this path so their control-plane calls do not
// depend on a gateway route that may be stale after an in-place upgrade.
func GetInternalTimerSchedulerURL() string {
	cfg := config.GetTimerSchedulerConfig()
	host := strings.TrimSpace(cfg.GetListenHost())
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, strconv.Itoa(cfg.GetPort()))
}

// BuildInternalTimerSchedulerURL builds a direct core-to-timer URL.
func BuildInternalTimerSchedulerURL(path string) string {
	return joinURL(GetInternalTimerSchedulerURL(), path)
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

func gatewayBaseURL() string {
	if configured := os.Getenv("GATEWAY_URL"); configured != "" {
		return configured
	}
	return config.GetGatewayURL()
}
