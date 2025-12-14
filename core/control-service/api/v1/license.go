package v1

import (
	"github.com/ai-agent-os/ai-agent-os/core/control-service/service"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/response"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/gin-gonic/gin"
)

// License License相关API
type License struct {
	licenseService *service.LicenseService
}

// NewLicense 创建License API（依赖注入）
func NewLicense(licenseService *service.LicenseService) *License {
	return &License{
		licenseService: licenseService,
	}
}

// GetStatus 获取 License 状态
// @Summary 获取 License 状态
// @Description 获取当前 License 的状态信息
// @Tags License管理
// @Accept json
// @Produce json
// @Success 200 {object} service.LicenseStatus "License状态"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/control/license/status [get]
func (l *License) GetStatus(c *gin.Context) {
	status := l.licenseService.GetStatus()
	response.OkWithData(c, status)
}

// Activate 激活 License（上传License文件并激活）
// @Summary 激活 License
// @Description 上传License文件并激活企业版
// @Tags License管理
// @Accept json
// @Produce json
// @Param license body string true "License文件内容（JSON格式）"
// @Success 200 {object} map[string]interface{} "激活成功"
// @Failure 400 {string} string "请求参数错误"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/control/license/activate [post]
func (l *License) Activate(c *gin.Context) {
	// 读取请求体（License文件内容）
	licenseData, err := c.GetRawData()
	if err != nil {
		logger.Errorf(c, "[License API] Failed to read license data: %v", err)
		response.FailWithMessage(c, "failed to read license data")
		return
	}

	if len(licenseData) == 0 {
		response.FailWithMessage(c, "license data is empty")
		return
	}

	// 激活License
	logger.Infof(c, "[License API] 开始激活 License...")
	if err := l.licenseService.Activate(c, licenseData); err != nil {
		logger.Errorf(c, "[License API] Failed to activate license: %v", err)
		response.FailWithMessage(c, err.Error())
		return
	}

	logger.Infof(c, "[License API] License 激活完成，获取状态...")
	// 返回激活状态
	status := l.licenseService.GetStatus()

	// 打印激活信息
	logger.Infof(c, "[License API] ========================================")
	logger.Infof(c, "[License API] License 激活成功！")
	logger.Infof(c, "[License API] ========================================")
	if status.IsValid {
		logger.Infof(c, "[License API] 版本: %s", status.Edition)
		if status.Customer != "" {
			logger.Infof(c, "[License API] 客户: %s", status.Customer)
		}
		if status.Description != "" {
			logger.Infof(c, "[License API] 描述: %s", status.Description)
		}
		if status.ExpiresAt != nil {
			logger.Infof(c, "[License API] 过期时间: %v", status.ExpiresAt)
		}
		if status.Features != nil {
			logger.Infof(c, "[License API] 功能: %+v", status.Features)
		}
		logger.Infof(c, "[License API] ========================================")
	}

	logger.Infof(c, "[License API] 准备返回响应...")
	response.OkWithData(c, status, map[string]interface{}{
		"message": "license activated successfully",
	})
	logger.Infof(c, "[License API] 响应已返回")
}

// Deactivate 注销 License（删除激活信息，回到社区版）
// @Summary 注销 License
// @Description 注销当前 License，删除激活信息，系统回到社区版（主要用于测试）
// @Tags License管理
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "注销成功"
// @Failure 500 {string} string "服务器内部错误"
// @Router /api/v1/control/license/deactivate [post]
func (l *License) Deactivate(c *gin.Context) {
	logger.Infof(c, "[License API] 开始注销 License...")
	if err := l.licenseService.Deactivate(c); err != nil {
		logger.Errorf(c, "[License API] Failed to deactivate license: %v", err)
		response.FailWithMessage(c, err.Error())
		return
	}

	logger.Infof(c, "[License API] License 注销完成，获取状态...")
	// 返回注销后的状态（应该是社区版）
	status := l.licenseService.GetStatus()

	logger.Infof(c, "[License API] ========================================")
	logger.Infof(c, "[License API] License 注销成功！")
	logger.Infof(c, "[License API] ========================================")
	logger.Infof(c, "[License API] 当前版本: %s", status.Edition)
	logger.Infof(c, "[License API] 是否社区版: %v", status.IsCommunity)
	logger.Infof(c, "[License API] ========================================")

	response.OkWithData(c, status, map[string]interface{}{
		"message": "license deactivated successfully",
	})
	logger.Infof(c, "[License API] 响应已返回")
}
