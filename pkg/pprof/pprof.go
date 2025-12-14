package pprof

import (
	"net/http"
	"net/http/pprof"

	"github.com/gin-gonic/gin"
)

// RegisterPprofRoutes 为 gin 引擎注册 pprof 路由
// 仅在开发环境或配置启用时使用
func RegisterPprofRoutes(router *gin.Engine) {
	RegisterPprofRoutesWithPrefix(router, "/debug/pprof")
}

// RegisterPprofRoutesWithPrefix 为 gin 引擎注册 pprof 路由（带前缀）
// serviceName: 服务名称，用于区分不同服务的 pprof 端点
// 例如：RegisterPprofRoutesWithPrefix(router, "/debug/pprof/app-server")
func RegisterPprofRoutesWithPrefix(router *gin.Engine, prefix string) {
	// 创建 pprof 路由组
	pprofGroup := router.Group(prefix)
	{
		// 主页面：显示所有可用的 profile
		pprofGroup.GET("/", gin.WrapH(http.HandlerFunc(pprof.Index)))

		// CPU profile：采集 CPU 使用情况
		pprofGroup.GET("/profile", gin.WrapH(http.HandlerFunc(pprof.Profile)))

		// Heap profile：采集内存使用情况
		pprofGroup.GET("/heap", gin.WrapH(http.HandlerFunc(pprof.Handler("heap").ServeHTTP)))

		// Goroutine profile：显示所有 goroutine 的堆栈
		pprofGroup.GET("/goroutine", gin.WrapH(http.HandlerFunc(pprof.Handler("goroutine").ServeHTTP)))

		// Block profile：显示阻塞操作
		pprofGroup.GET("/block", gin.WrapH(http.HandlerFunc(pprof.Handler("block").ServeHTTP)))

		// Mutex profile：显示互斥锁竞争
		pprofGroup.GET("/mutex", gin.WrapH(http.HandlerFunc(pprof.Handler("mutex").ServeHTTP)))

		// Allocs profile：显示内存分配
		pprofGroup.GET("/allocs", gin.WrapH(http.HandlerFunc(pprof.Handler("allocs").ServeHTTP)))

		// Cmdline：显示命令行参数
		pprofGroup.GET("/cmdline", gin.WrapH(http.HandlerFunc(pprof.Cmdline)))

		// Symbol：符号查找
		pprofGroup.GET("/symbol", gin.WrapH(http.HandlerFunc(pprof.Symbol)))

		// Trace：执行追踪
		pprofGroup.GET("/trace", gin.WrapH(http.HandlerFunc(pprof.Trace)))
	}
}

