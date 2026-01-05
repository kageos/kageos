package response

// HTTP 状态码常量
const (
	// StatusUnauthorized 401 - 未授权（需要重新登录）
	// 前端处理：自动跳转到登录页面
	StatusUnauthorized = 401
	// StatusForbidden 403 - 权限不足（需要申请权限）
	// 前端处理：显示权限申请页面
	StatusForbidden = 403
)

// 响应错误码常量（字符串类型，用于前端识别不同的错误场景）
// 注意：前端会根据 HTTP 状态码（401/403）执行不同的处理逻辑
// 这些 code 主要用于日志记录和错误分类，前端可以根据 code 做更细粒度的处理
const (
	// CodeTokenExpired Token 已过期（需要重新登录）
	// HTTP 状态码：401
	// 前端处理：自动跳转到登录页面
	CodeTokenExpired = "TOKEN_EXPIRED"
	
	// CodeTokenInvalid Token 无效（需要重新登录）
	// HTTP 状态码：401
	// 前端处理：自动跳转到登录页面
	CodeTokenInvalid = "TOKEN_INVALID"
	
	// CodeTokenBlacklisted Token 已失效（被加入黑名单，需要重新登录）
	// HTTP 状态码：401
	// 前端处理：自动跳转到登录页面
	CodeTokenBlacklisted = "TOKEN_BLACKLISTED"
	
	// CodePermissionDenied 权限不足（需要申请权限）
	// HTTP 状态码：403
	// 前端处理：显示权限申请页面
	CodePermissionDenied = "PERMISSION_DENIED"
)

// GetTokenExpiredResponse 获取 Token 过期响应（用于前端跳转登录页）
// HTTP 状态码：401
// 返回格式：{"code": "TOKEN_EXPIRED", "msg": "Token 已过期，请重新登录", "data": null}
func GetTokenExpiredResponse() map[string]interface{} {
	return map[string]interface{}{
		"code": CodeTokenExpired,
		"msg":  "Token 已过期，请重新登录",
		"data": nil,
	}
}

// GetTokenInvalidResponse 获取 Token 无效响应（用于前端跳转登录页）
// HTTP 状态码：401
// 返回格式：{"code": "TOKEN_INVALID", "msg": "Token 无效，请重新登录", "data": null}
func GetTokenInvalidResponse() map[string]interface{} {
	return map[string]interface{}{
		"code": CodeTokenInvalid,
		"msg":  "Token 无效，请重新登录",
		"data": nil,
	}
}

// GetTokenBlacklistedResponse 获取 Token 黑名单响应（用于前端跳转登录页）
// HTTP 状态码：401
// 返回格式：{"code": "TOKEN_BLACKLISTED", "msg": "Token 已失效，请重新登录", "data": null}
func GetTokenBlacklistedResponse() map[string]interface{} {
	return map[string]interface{}{
		"code": CodeTokenBlacklisted,
		"msg":  "Token 已失效，请重新登录",
		"data": nil,
	}
}

