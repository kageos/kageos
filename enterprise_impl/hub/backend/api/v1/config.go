package v1

import (
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/response"
	"github.com/gin-gonic/gin"
)

// Config 前端所需配置（主站地址等）
type Config struct{}

// NewConfig 创建 Config 处理器
func NewConfig() *Config {
	return &Config{}
}

// GetConfig 获取 Hub 前端配置（公开接口，无需认证）
// 用于「试用」按钮跳转：主站前端地址 + /workspace + 目录 full_code_path（主站前端≠主站后端）
// @Summary 获取 Hub 前端配置
// @Description 返回主站前端地址，供试用跳转使用（开发 5173，线上 8999 等）
// @Tags Hub配置
// @Produce json
// @Success 200 {object} map[string]string "main_site_url 为主站前端地址"
// @Router /api/v1/config [get]
func (h *Config) GetConfig(c *gin.Context) {
	cfg := config.GetHubConfig()
	response.OkWithData(c, map[string]string{
		"main_site_url": cfg.GetMainSiteURL(),
	})
}
