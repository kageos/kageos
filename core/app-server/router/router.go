package router

import (
	"time"

	middleware2 "github.com/ai-agent-os/ai-agent-os/pkg/middleware"

	v1 "github.com/ai-agent-os/ai-agent-os/core/app-server/api/v1"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

var start time.Time

func Init() *gin.Engine {
	start = time.Now()
	// 创建gin引擎
	r := gin.New()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":    "ok",
			"timestamp": time.Now().Format(time.DateTime),
			"uptime":    time.Since(start).String(),
		})
	})

	// Swagger 文档路由
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.Use(gin.Recovery())
	r.Use(middleware2.Cors()) // 添加 CORS 中间件
	r.Use(middleware2.WithTraceId())

	apiV1 := r.Group("/api/v1")

	// 认证相关路由（不需要JWT验证）
	auth := apiV1.Group("/auth")
	newAuth := v1.NewAuth()
	auth.POST("/send_email_code", newAuth.SendEmailCode)
	auth.POST("/register", newAuth.Register)
	auth.POST("/login", newAuth.Login)
	auth.POST("/refresh", newAuth.RefreshToken)
	auth.POST("/logout", newAuth.Logout)

	// 应用管理路由（需要JWT验证）
	app := apiV1.Group("/app")
	app.Use(middleware2.JWTAuth()) // 应用管理需要JWT认证
	newApp := v1.NewDefaultApp()
	app.POST("/create", newApp.CreateApp)
	app.POST("/update/:app", newApp.UpdateApp)
	app.DELETE("/delete/:app", newApp.DeleteApp)
	// 支持所有 HTTP 方法的请求应用接口
	app.Any("/request/:app/*router", newApp.RequestApp)

	return r
}
