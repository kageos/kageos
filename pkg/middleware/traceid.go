package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// WithTraceId 为请求添加跟踪ID的中间件
func WithTraceId() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("trace_id", uuid.New().String())
	}
}
