package middleware

import (
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/enterprise"
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/response"
	"github.com/ai-agent-os/ai-agent-os/pkg/license"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/gin-gonic/gin"
)

// RequireFeature 统一的功能鉴权中间件
// 用于保护企业版功能接口，只有激活了企业版 License 且支持该功能的用户才能访问
//
// 参数：
//   - feature: 功能名称（使用 enterprise.FeatureXXX 常量）
//
// 使用示例：
//   operateLog := apiV1.Group("/operate_log")
//   operateLog.Use(middleware.RequireFeature(enterprise.FeatureOperateLog))
//
// 说明：
//   - 替代原有的 OperateLogAuth()、OrganizationAuth() 等中间件
//   - 统一错误处理和日志记录
//   - 支持所有企业版功能
func RequireFeature(feature string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取 License 管理器
		licenseMgr := license.GetManager()

		// 检查是否有该功能（企业版功能）
		if !licenseMgr.HasFeature(feature) {
			// 获取功能名称（用于错误提示）
			featureName := getFeatureDisplayName(feature)
			
			logger.Warnf(c, "[RequireFeature] 访问功能被拒绝：功能=%s，原因=未激活企业版 License 或 License 不支持该功能", feature)
			response.FailWithMessage(c, fmt.Sprintf("此功能需要企业版 License：%s，请升级到企业版", featureName))
			c.Abort()
			return
		}

		logger.Debugf(c, "[RequireFeature] 功能鉴权通过：功能=%s", feature)
		c.Next()
	}
}

// getFeatureDisplayName 获取功能显示名称（用于错误提示）
func getFeatureDisplayName(feature string) string {
	switch feature {
	case enterprise.FeatureOperateLog:
		return "操作日志"
	case enterprise.FeatureOrganization:
		return "组织架构"
	case enterprise.FeaturePermission:
		return "权限管理"
	case enterprise.FeatureWorkflow:
		return "工作流"
	case enterprise.FeatureApproval:
		return "审批流程"
	case enterprise.FeatureScheduledTask:
		return "定时任务"
	case enterprise.FeatureRecycleBin:
		return "回收站"
	case enterprise.FeatureChangeLog:
		return "变更日志"
	case enterprise.FeatureNotification:
		return "通知中心"
	case enterprise.FeatureConfigManagement:
		return "配置管理"
	case enterprise.FeatureQuickLink:
		return "快链"
	default:
		return feature
	}
}

// OperateLogAuth 操作日志功能鉴权中间件（向后兼容）
// 已废弃：请使用 RequireFeature(enterprise.FeatureOperateLog) 替代
//
// Deprecated: 使用 RequireFeature(enterprise.FeatureOperateLog) 替代
func OperateLogAuth() gin.HandlerFunc {
	return RequireFeature(enterprise.FeatureOperateLog)
}

