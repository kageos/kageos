// @title AI Agent OS API
// @version 1.0
// @description AI Agent OS 应用管理平台 API 文档
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:9090
// @BasePath /api/v1

// @securityDefinitions.basic BasicAuth

// @externalDocs.description  OpenAPI
// @externalDocs.url          https://swagger.io/resources/open-api/
package main

import (
	"fmt"
	"log"
	"os"

	_ "github.com/ai-agent-os/ai-agent-os/core/app-server/docs"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/upstrem"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/global"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/router"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	
	// 导入认证 API 以确保 swag 扫描到
	_ "github.com/ai-agent-os/ai-agent-os/core/app-server/api/v1"
)

func main() {
	// 获取配置
	cfg := config.GetAppServerConfig()

	// 初始化数据库
	model.InitDB()
	defer model.CloseDB()

	// 初始化全局变量（NATS）
	global.Init()
	defer global.Close()

	upstrem.Init()

	// 初始化日志系统
	logConfig := logger.Config{
		Level:      cfg.GetLogLevel(),
		Filename:   "./logs/app-server.log",
		MaxSize:    100,
		MaxBackups: 3,
		MaxAge:     7,
		Compress:   true,
		IsDev:      cfg.IsDebug(),
	}

	if err := logger.Init(logConfig); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	r := router.Init()
	port := fmt.Sprintf(":%d", cfg.GetPort())
	log.Fatal(r.Run(port))
}
