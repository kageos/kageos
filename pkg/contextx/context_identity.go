package contextx

import (
	"context"

	"github.com/kageos/kageos/pkg/logger"

	"github.com/gin-gonic/gin"
)

// GetTraceId 获取追踪ID
// ⭐ 只从 HTTP Header 读取（统一方式，避免混乱）
// 支持从 *gin.Context 或标准 context.Context 读取
func GetTraceId(c context.Context) string {
	v, ok := c.(*gin.Context)
	if ok {
		// ✨ 只从 HTTP header 读取
		return v.GetHeader(TraceIdHeader)
	}

	// 从标准 context.Value 读取（可能是 ToContext 转换后的标准 context，或 context.WithValue 包装的）
	if value := c.Value(TraceIdHeader); value != nil {
		if traceId, ok := value.(string); ok && traceId != "" {
			return traceId
		}
	}

	return ""
}

// GetRequestUser 获取请求用户
// ⭐ 只从 HTTP Header 读取（统一方式，避免混乱）
// 支持从 *gin.Context 或标准 context.Context 读取
func GetRequestUser(c context.Context) string {
	// 首先尝试转换为 *gin.Context（可以读取 header）
	v, ok := c.(*gin.Context)
	if ok {
		// ✨ 只从 HTTP header 读取
		requestUser := v.GetHeader(RequestUserHeader)
		if requestUser == "" {
			// ⭐ 如果 header 为空，打印警告日志（包含更多调试信息）
			token := v.GetHeader(TokenHeader)
			logger.Warnf(c, "[GetRequestUser] 无法获取 RequestUser - Path: %s, IP: %s, HasToken: %v, TokenLength: %d, X-Request-User Header: %s",
				v.Request.URL.Path, v.ClientIP(), token != "", len(token), v.GetHeader(RequestUserHeader))
		}
		return requestUser
	}

	// 从标准 context.Value 读取（可能是 ToContext 转换后的标准 context，或 context.WithValue 包装的）
	if value := c.Value(RequestUserHeader); value != nil {
		if requestUser, ok := value.(string); ok && requestUser != "" {
			return requestUser
		}
	}

	return ""
}

// GetRequestDepartmentFullPath 获取请求用户的组织架构路径
// ⭐ 只从 HTTP Header 读取（统一方式，避免混乱）
// 支持从 *gin.Context 或标准 context.Context 读取
func GetRequestDepartmentFullPath(c context.Context) string {
	// 首先尝试转换为 *gin.Context（可以读取 header）
	v, ok := c.(*gin.Context)
	if ok {
		// ✨ 只从 HTTP header 读取
		return v.GetHeader(DepartmentFullPathHeader)
	}

	// 从标准 context.Value 读取（可能是 ToContext 转换后的标准 context，或 context.WithValue 包装的）
	if value := c.Value(DepartmentFullPathHeader); value != nil {
		if deptPath, ok := value.(string); ok && deptPath != "" {
			return deptPath
		}
	}

	return ""
}

func GetRequestCompanyCode(c context.Context) string {
	return getStringFromContextOrHeader(c, CompanyCodeHeader)
}

func GetRequestCompanyName(c context.Context) string {
	return getStringFromContextOrHeader(c, CompanyNameHeader)
}

func GetRequestCompanyLogoURL(c context.Context) string {
	return getStringFromContextOrHeader(c, CompanyLogoURLHeader)
}

func GetRequestUserID(c context.Context) string {
	return getStringFromContextOrHeader(c, UserIDHeader)
}

func GetRequestUserEmail(c context.Context) string {
	return getStringFromContextOrHeader(c, UserEmailHeader)
}

func GetRequestLeaderUsername(c context.Context) string {
	return getStringFromContextOrHeader(c, LeaderUsernameHeader)
}

func getStringFromContextOrHeader(c context.Context, key string) string {
	if v, ok := c.(*gin.Context); ok {
		return v.GetHeader(key)
	}
	if value := c.Value(key); value != nil {
		if s, ok := value.(string); ok && s != "" {
			return s
		}
	}
	return ""
}

// GetToken 获取认证 Token
// ⭐ 只从 HTTP Header 读取（统一方式，避免混乱）
func GetToken(c context.Context) string {
	v, ok := c.(*gin.Context)
	if ok {
		// ✨ 只从 HTTP header 读取
		return v.GetHeader(TokenHeader)
	}
	// 从标准 context.Value 读取（可能是 ToContext 转换后的标准 context，或 context.WithValue 包装的）
	if value := c.Value(TokenHeader); value != nil {
		if token, ok := value.(string); ok && token != "" {
			return token
		}
	}
	return ""
}

// WithRequestUser 注入请求用户到 context（用于后台任务等无 HTTP 请求场景）
func WithRequestUser(ctx context.Context, username string) context.Context {
	if username == "" {
		return ctx
	}
	return context.WithValue(ctx, RequestUserHeader, username)
}

// WithToken 注入 Token 到 context
func WithToken(ctx context.Context, token string) context.Context {
	if token == "" {
		return ctx
	}
	return context.WithValue(ctx, TokenHeader, token)
}

// WithDepartmentFullPath 注入部门路径到 context
func WithDepartmentFullPath(ctx context.Context, deptPath string) context.Context {
	if deptPath == "" {
		return ctx
	}
	return context.WithValue(ctx, DepartmentFullPathHeader, deptPath)
}

// WithTraceId 注入 TraceId 到 context
func WithTraceId(ctx context.Context, traceId string) context.Context {
	if traceId == "" {
		return ctx
	}
	return context.WithValue(ctx, TraceIdHeader, traceId)
}
