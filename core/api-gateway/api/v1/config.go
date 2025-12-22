package v1

import (
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/gin-gonic/gin"
)

// Config 配置处理器
type Config struct {
}

// NewConfig 创建配置处理器
func NewConfig() *Config {
	return &Config{}
}

// ServiceConfigResponse 服务配置响应
type ServiceConfigResponse struct {
	Gateway  string             `json:"gateway"`  // 网关地址
	Services map[string]Service `json:"services"` // 服务配置
	Version  string             `json:"version"`  // 版本
}

// Service 服务配置
type Service struct {
	URL       string `json:"url"`        // 通过网关的地址
	Path      string `json:"path"`       // 路径前缀
	DirectURL string `json:"direct_url"` // 直接地址（用于 SDK 内部调用，绕过网关）
}

// GetConfig 获取配置接口
// @Summary 获取服务配置
// @Description 返回所有服务的访问地址（通过网关和直接地址）
// @Tags config
// @Produce json
// @Success 200 {object} ServiceConfigResponse
// @Router /api/v1/config [get]
func (c *Config) GetConfig(ctx *gin.Context) {
	cfg := config.GetAPIGatewayConfig()

	// 获取网关地址（从全局配置读取，裸机服务访问）
	globalConfig := config.GetGlobalSharedConfig()
	gatewayURL := globalConfig.Gateway.GetBaseURL()

	// 构建服务配置
	services := make(map[string]Service)

	// 遍历路由配置，提取服务信息
	for _, route := range cfg.Routes {
		if len(route.Targets) == 0 || route.ServiceName == "" {
			continue
		}

		// 获取直接地址（第一个 target）
		directURL := route.Targets[0].URL

		// 构建服务配置
		services[route.ServiceName] = Service{
			URL:       gatewayURL, // 通过网关访问
			Path:      route.Path,  // 路径前缀
			DirectURL: directURL,   // 直接地址（SDK 内部调用）
		}
	}

	// 返回详细的服务配置
	ctx.JSON(200, ServiceConfigResponse{
		Gateway:  gatewayURL,
		Services: services,
		Version:  "v1.0.0",
	})
}

