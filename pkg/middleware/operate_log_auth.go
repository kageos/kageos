package middleware

import (
	"github.com/ai-agent-os/ai-agent-os/pkg/ginx/response"
	"github.com/ai-agent-os/ai-agent-os/pkg/license"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/gin-gonic/gin"
)

// OperateLogAuth 操作日志功能鉴权中间件
// 用于保护操作日志查询接口，只有激活了企业版 License 且支持操作日志功能的用户才能访问
func OperateLogAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取 License 管理器
		licenseMgr := license.GetManager()

		// 检查是否有操作日志功能（企业版功能）
		if !licenseMgr.HasOperateLogFeature() {
			logger.Warnf(c, "[OperateLogAuth] 访问操作日志功能被拒绝：未激活企业版 License 或 License 不支持操作日志功能")
			response.FailWithMessage(c, "此功能需要企业版 License，请升级到企业版")
			c.Abort()
			return
		}

		logger.Infof(c, "[OperateLogAuth] 操作日志功能鉴权通过")
		c.Next()
	}
}

