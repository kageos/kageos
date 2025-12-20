package middleware

import (
	"compress/gzip"
	"io"
	"strings"

	"github.com/gin-gonic/gin"
)

// Gzip 压缩中间件
// 自动压缩响应内容，减少传输数据大小
func Gzip() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查客户端是否支持 gzip
		if !strings.Contains(c.GetHeader("Accept-Encoding"), "gzip") {
			c.Next()
			return
		}

		// 创建 gzip writer
		gz := gzip.NewWriter(c.Writer)
		defer gz.Close()

		// 设置响应头（必须在写入响应之前设置）
		c.Writer.Header().Set("Content-Encoding", "gzip")
		c.Writer.Header().Set("Vary", "Accept-Encoding")

		// 使用 gzip writer 包装响应
		c.Writer = &gzipWriter{
			ResponseWriter: c.Writer,
			Writer:         gz,
		}

		c.Next()
	}
}

// gzipWriter 包装 gin.ResponseWriter，将内容写入 gzip writer
type gzipWriter struct {
	gin.ResponseWriter
	Writer io.Writer
}

func (g *gzipWriter) Write(data []byte) (int, error) {
	return g.Writer.Write(data)
}

func (g *gzipWriter) WriteString(s string) (int, error) {
	return g.Writer.Write([]byte(s))
}
