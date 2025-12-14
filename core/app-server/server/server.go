package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/enterprise"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/service"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/license"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	middleware2 "github.com/ai-agent-os/ai-agent-os/pkg/middleware"
	"github.com/ai-agent-os/ai-agent-os/pkg/subjects"
	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

// Server app-server 服务器
type Server struct {
	// 配置
	cfg *config.AppServerConfig

	// 核心组件
	db         *gorm.DB
	natsConn   *nats.Conn
	httpServer *gin.Engine

	// 服务
	appService         *service.AppService
	authService        *service.AuthService
	emailService       *service.EmailService
	jwtService         *service.JWTService
	appRuntime         *service.AppRuntime
	serviceTreeService *service.ServiceTreeService
	functionService    *service.FunctionService
	userService        *service.UserService
	operateLogService  *service.OperateLogService

	// 上游服务
	natsService *service.NatsService

	// NATS 订阅
	functionGenSub *nats.Subscription

	// 上下文
	ctx context.Context

	//企业功能
	operateLogger enterprise.OperateLogger

	// License Client
	licenseClient *license.Client
}

// NewServer 创建新的服务器实例
func NewServer(cfg *config.AppServerConfig) (*Server, error) {
	ctx := context.Background()

	s := &Server{
		cfg: cfg,
		ctx: ctx,
	}

	// ⭐ 1. 首先加载 License（必须在其他初始化之前）
	if err := s.initLicense(ctx); err != nil {
		// License 加载失败，记录警告但不中断启动（社区版可以继续运行）
		logger.Warnf(ctx, "[Server] Failed to load license: %v, continuing with community edition", err)
	}

	// 初始化各个组件
	if err := s.initDatabase(ctx); err != nil {
		return nil, fmt.Errorf("failed to init database: %w", err)
	}

	if err := s.initNATS(ctx); err != nil {
		return nil, fmt.Errorf("failed to init NATS: %w", err)
	}

	// ⭐ 初始化 License Client（在 NATS 初始化之后）
	if err := s.initLicenseClient(ctx); err != nil {
		// License Client 初始化失败，记录警告但不中断启动（社区版可以继续运行）
		logger.Warnf(ctx, "[Server] Failed to init license client: %v, continuing with community edition", err)
	}

	if err := s.initServices(ctx); err != nil {
		return nil, fmt.Errorf("failed to init services: %w", err)
	}

	// ⭐ 2. 初始化企业功能（在数据库和 NATS 初始化之后）
	if err := s.initEnterprise(); err != nil {
		return nil, fmt.Errorf("failed to init enterprise features: %w", err)
	}

	if err := s.initRouter(ctx); err != nil {
		return nil, fmt.Errorf("failed to init router: %w", err)
	}

	// 初始化 NATS 订阅
	if err := s.initNATSSubscriptions(ctx); err != nil {
		return nil, fmt.Errorf("failed to init NATS subscriptions: %w", err)
	}

	return s, nil
}

// Start 启动服务器
func (s *Server) Start(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Starting app-server...")

	// 启动 HTTP 服务器
	port := fmt.Sprintf(":%d", s.cfg.GetPort())
	logger.Infof(ctx, "[Server] HTTP server starting on port %s", port)

	go func() {
		if err := s.httpServer.Run(port); err != nil {
			logger.Errorf(ctx, "[Server] HTTP server error: %v", err)
		}
	}()

	logger.Infof(ctx, "[Server] App-server started successfully")
	logger.Infof(ctx, "[Server] NATS subscriptions are active")
	return nil
}

// Stop 停止服务器（优雅关闭）
func (s *Server) Stop(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Stopping app-server...")

	// 关闭 NATS 订阅
	if s.functionGenSub != nil {
		if err := s.functionGenSub.Unsubscribe(); err != nil {
			logger.Warnf(ctx, "[Server] Failed to unsubscribe function gen: %v", err)
		} else {
			logger.Infof(ctx, "[Server] Function gen subscription closed")
		}
	}

	// 关闭 AppRuntime 服务（包括 NATS 订阅）
	if s.appRuntime != nil {
		s.appRuntime.Close()
		logger.Infof(ctx, "[Server] AppRuntime service closed")
	}

	// 关闭 NATS 服务
	if s.natsService != nil {
		s.natsService.Close()
		logger.Infof(ctx, "[Server] NATS service closed")
	}

	// 关闭 License Client
	if s.licenseClient != nil {
		if err := s.licenseClient.Stop(ctx); err != nil {
			logger.Warnf(ctx, "[Server] Failed to stop license client: %v", err)
		} else {
			logger.Infof(ctx, "[Server] License client stopped")
		}
	}

	// 关闭 NATS 连接
	if s.natsConn != nil {
		s.natsConn.Close()
		logger.Infof(ctx, "[Server] NATS connection closed")
	}

	// 关闭数据库连接
	if s.db != nil {
		sqlDB, err := s.db.DB()
		if err == nil {
			sqlDB.Close()
			logger.Infof(ctx, "[Server] Database connection closed")
		}
	}

	logger.Infof(ctx, "[Server] App-server stopped")
	return nil
}

// initLicense 初始化 License（从文件加载，向后兼容）
// 在服务器启动时加载和验证 License 文件
func (s *Server) initLicense(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Initializing license from file...")

	// 获取 License 管理器
	licenseMgr := license.GetManager()

	// 加载 License（如果文件不存在，返回 nil，表示社区版）
	if err := licenseMgr.LoadLicense(""); err != nil {
		// License 加载失败，可能是文件不存在（社区版）或验证失败
		// 如果是验证失败，记录错误但不中断启动（允许降级到社区版）
		logger.Warnf(ctx, "[Server] License loading from file failed: %v", err)
		return err
	}

	// 检查 License 状态
	currentLicense := licenseMgr.GetLicense()
	if currentLicense == nil {
		logger.Infof(ctx, "[Server] Community edition (no license file)")
	} else {
		logger.Infof(ctx, "[Server] License loaded from file: Edition=%s, Customer=%s, ExpiresAt=%v",
			currentLicense.Edition, currentLicense.Customer, currentLicense.ExpiresAt)
	}

	return nil
}

// initLicenseClient 初始化 License Client（通过 NATS 获取和刷新 License）
func (s *Server) initLicenseClient(ctx context.Context) error {
	// 检查是否启用 Control Service 客户端
	controlCfg := s.cfg.ControlService
	if !controlCfg.IsEnabled() {
		logger.Infof(ctx, "[Server] Control Service client is disabled, skipping license client initialization")
		return nil
	}

	// 检查加密密钥
	encryptionKey := controlCfg.GetEncryptionKey()
	if len(encryptionKey) != 32 {
		return fmt.Errorf("encryption key must be 32 bytes, got %d bytes", len(encryptionKey))
	}

	// 确定使用的 NATS 连接
	// 如果配置了独立的 NATS URL，需要创建新连接；否则使用现有的连接
	natsConn := s.natsConn
	if controlCfg.GetNatsURL() != "" {
		// 使用独立的 NATS 连接
		var err error
		natsConn, err = nats.Connect(controlCfg.GetNatsURL())
		if err != nil {
			return fmt.Errorf("failed to connect to Control Service NATS: %w", err)
		}
		logger.Infof(ctx, "[Server] Connected to Control Service NATS: %s", controlCfg.GetNatsURL())
	}

	// 创建 License Client
	client, err := license.NewClient(natsConn, encryptionKey, controlCfg.GetKeyPath())
	if err != nil {
		return fmt.Errorf("failed to create license client: %w", err)
	}

	// 启动 License Client
	if err := client.Start(ctx); err != nil {
		return fmt.Errorf("failed to start license client: %w", err)
	}

	s.licenseClient = client
	logger.Infof(ctx, "[Server] License client initialized successfully")
	return nil
}

// initDatabase 初始化数据库
func (s *Server) initDatabase(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Initializing database...")

	dbCfg := s.cfg.DB

	// 配置 GORM 日志
	gormConfig := &gorm.Config{}

	// 如果启用了数据库日志
	if dbCfg.LogLevel != "silent" {
		var logLevel gormLogger.LogLevel
		switch dbCfg.LogLevel {
		case "error":
			logLevel = gormLogger.Error
		case "warn":
			logLevel = gormLogger.Warn
		case "info":
			logLevel = gormLogger.Info
		default:
			logLevel = gormLogger.Warn
		}

		// 配置慢查询阈值
		slowThreshold := time.Duration(dbCfg.SlowThreshold) * time.Millisecond
		if slowThreshold == 0 {
			slowThreshold = 200 * time.Millisecond // 默认200毫秒
		}

		// 使用 GORM 默认日志配置
		gormConfig.Logger = gormLogger.Default.LogMode(logLevel)
	} else {
		// 禁用日志
		gormConfig.Logger = gormLogger.Default.LogMode(gormLogger.Silent)
	}

	var err error
	switch dbCfg.Type {
	case "mysql":
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			dbCfg.User, dbCfg.Password, dbCfg.Host, dbCfg.Port, dbCfg.Name)
		s.db, err = gorm.Open(mysql.Open(dsn), gormConfig)
		if err != nil {
			return fmt.Errorf("failed to connect to MySQL: %w", err)
		}
	default:
		return fmt.Errorf("unsupported database type: %s", dbCfg.Type)
	}

	// 自动迁移表结构
	if err := model.InitTables(s.db); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	logger.Infof(ctx, "[Server] Database initialized successfully")
	return nil
}

// initNATS 初始化 NATS 连接
func (s *Server) initNATS(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Initializing NATS connection...")

	var err error
	s.natsConn, err = nats.Connect(s.cfg.Nats.URL)
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}

	logger.Infof(ctx, "[Server] NATS connection initialized successfully")
	return nil
}

// initServices 初始化所有业务服务
func (s *Server) initServices(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Initializing services...")

	// 初始化 NATS 服务 - 其他服务的基础依赖
	s.natsService = service.NewNatsServiceWithDB(s.db)

	// 初始化应用运行时服务
	s.appRuntime = service.NewAppRuntimeService(s.cfg, s.natsService)

	// 初始化应用服务
	userRepo := repository.NewUserRepository(s.db)
	appRepo := repository.NewAppRepository(s.db)
	hostRepo := repository.NewHostRepository(s.db)
	userSessionRepo := repository.NewUserSessionRepository(s.db)
	functionRepo := repository.NewFunctionRepository(s.db)
	serviceTreeRepo := repository.NewServiceTreeRepository(s.db)
	sourceCodeRepo := repository.NewSourceCodeRepository(s.db)
	operateLogRepo := repository.NewOperateLogRepository(s.db)

	s.appService = service.NewAppService(s.appRuntime, userRepo, appRepo, functionRepo, serviceTreeRepo, sourceCodeRepo, operateLogRepo)

	// 初始化认证服务
	s.authService = service.NewAuthService(userRepo, hostRepo, userSessionRepo)

	// 初始化邮件服务
	emailCodeRepo := repository.NewEmailCodeRepository(s.db)
	s.emailService = service.NewEmailService(emailCodeRepo)

	// 初始化 JWT 服务
	s.jwtService = service.NewJWTService()

	s.serviceTreeService = service.NewServiceTreeService(serviceTreeRepo, appRepo, s.appRuntime)

	// 初始化函数服务（需要更多依赖）
	s.functionService = service.NewFunctionService(functionRepo, sourceCodeRepo, appRepo, serviceTreeRepo, s.appRuntime, s.appService)

	// 初始化用户服务
	s.userService = service.NewUserService(userRepo)

	// 初始化操作日志服务
	s.operateLogService = service.NewOperateLogService(operateLogRepo)

	logger.Infof(ctx, "[Server] Services initialized successfully")
	return nil
}

// initRouter 初始化路由
func (s *Server) initRouter(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Initializing router...")

	// 创建 gin 引擎
	s.httpServer = gin.New()

	// 添加中间件
	s.httpServer.Use(gin.Recovery())
	s.httpServer.Use(middleware2.Cors())
	// ✅ 移除 WithTraceId 中间件，统一在网关生成 TraceId
	// s.httpServer.Use(middleware2.WithTraceId())

	// 设置路由
	s.setupRoutes()

	// 设置 router 引用

	logger.Infof(ctx, "[Server] Router initialized successfully")
	return nil
}

// healthHandler 健康检查处理器
func (s *Server) healthHandler(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":    "ok",
		"timestamp": time.Now().Format(time.DateTime),
		"service":   "app-server",
	})
}

// GetDB 获取数据库连接
func (s *Server) GetDB() *gorm.DB {
	return s.db
}

// GetNATS 获取 NATS 连接
func (s *Server) GetNATS() *nats.Conn {
	return s.natsConn
}

// GetAppService 获取应用服务
func (s *Server) GetAppService() *service.AppService {
	return s.appService
}

// GetAuthService 获取认证服务
func (s *Server) GetAuthService() *service.AuthService {
	return s.authService
}

// initNATSSubscriptions 初始化 NATS 订阅
func (s *Server) initNATSSubscriptions(ctx context.Context) error {
	logger.Infof(ctx, "[Server] Initializing NATS subscriptions...")

	// 订阅 agent-server 的函数生成结果主题
	subject := subjects.GetAgentServerFunctionGenSubject()
	sub, err := s.natsConn.Subscribe(subject, s.handleFunctionGenResult)
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", subject, err)
	}
	s.functionGenSub = sub
	logger.Infof(ctx, "[Server] Subscribed to %s", subject)

	return nil
}

// handleFunctionGenResult 处理函数生成结果消息
func (s *Server) handleFunctionGenResult(msg *nats.Msg) {
	ctx := context.Background()

	// 从消息 header 中获取 trace_id 和 user
	traceID := msg.Header.Get("X-Trace-Id")
	requestUser := msg.Header.Get("X-Request-User")

	// 如果有 trace_id，设置到 context 中
	if traceID != "" {
		ctx = context.WithValue(ctx, "trace_id", traceID)
	}
	if requestUser != "" {
		ctx = context.WithValue(ctx, "request_user", requestUser)
	}

	// 解析消息体
	var result dto.FunctionGenResult
	if err := json.Unmarshal(msg.Data, &result); err != nil {
		logger.Errorf(ctx, "[Server] Failed to unmarshal function gen result: %v, Data: %s", err, string(msg.Data))
		return
	}

	// 打印日志
	logger.Infof(ctx, "[Server] Received function generation result:")
	logger.Infof(ctx, "[Server]   RecordID: %d", result.RecordID)
	logger.Infof(ctx, "[Server]   AgentID: %d", result.AgentID)
	logger.Infof(ctx, "[Server]   TreeID: %d", result.TreeID)
	logger.Infof(ctx, "[Server]   User: %s", result.User)
	logger.Infof(ctx, "[Server]   Code length: %d bytes", len(result.Code))
	logger.Infof(ctx, "[Server]   TraceID: %s", traceID)
	logger.Infof(ctx, "[Server]   RequestUser: %s", requestUser)

	// 1. 根据 TreeID 获取 ServiceTree（需要预加载 App）
	serviceTreeRepo := repository.NewServiceTreeRepository(s.db)
	serviceTree, err := serviceTreeRepo.GetByID(result.TreeID)
	if err != nil {
		logger.Errorf(ctx, "[Server] 获取 ServiceTree 失败: TreeID=%d, error=%v", result.TreeID, err)
		return
	}

	// 预加载 App 信息（如果还没有加载）
	if serviceTree.App == nil {
		appRepo := repository.NewAppRepository(s.db)
		app, err := appRepo.GetAppByID(serviceTree.AppID)
		if err != nil {
			logger.Errorf(ctx, "[Server] 获取 App 失败: AppID=%d, error=%v", serviceTree.AppID, err)
			return
		}
		serviceTree.App = app
	}

	// 2. 从 ServiceTree 中提取 package 路径（使用 model 方法）
	packagePath := serviceTree.GetPackagePathForFileCreation()

	// 3. 从 LLM 响应中提取代码（可能包含 Markdown 代码块）
	extractedCode := s.extractCodeFromMarkdown(result.Code)
	logger.Infof(ctx, "[Server] 代码提取完成 - 原始长度: %d, 提取后长度: %d", len(result.Code), len(extractedCode))

	// 4. 从生成的代码中解析 group_code（文件名）
	// 优先从代码中的 GroupCode 字段提取，其次从文件名注释，最后从结构体名称推断
	groupCode := s.extractGroupCodeFromCode(extractedCode)
	if groupCode == "" {
		logger.Errorf(ctx, "[Server] 无法从代码中提取 group_code，使用 ServiceTree.Code 作为 fallback: %s", serviceTree.Code)
		groupCode = serviceTree.Code
	}

	logger.Infof(ctx, "[Server] 提取信息: Package=%s, GroupCode=%s", packagePath, groupCode)

	// 5. 构建 CreateFunctionInfo
	createFunction := &dto.CreateFunctionInfo{
		Package:    packagePath,
		GroupCode:  groupCode,
		SourceCode: extractedCode, // 使用提取后的代码
	}

	// 4. 调用 AppService.UpdateApp，传入 CreateFunctions
	updateReq := &dto.UpdateAppReq{
		User:            result.User,
		App:             serviceTree.App.Code,
		CreateFunctions: []*dto.CreateFunctionInfo{createFunction},
	}

	logger.Infof(ctx, "[Server] 调用 AppService.UpdateApp: User=%s, App=%s, Package=%s, GroupCode=%s",
		updateReq.User, updateReq.App, packagePath, groupCode)

	updateResp, err := s.appService.UpdateApp(ctx, updateReq)
	if err != nil {
		logger.Errorf(ctx, "[Server] AppService.UpdateApp 失败: error=%v", err)
		return
	}

	logger.Infof(ctx, "[Server] 函数创建成功: Package=%s, GroupCode=%s", packagePath, groupCode)

	// 6. 获取新增的 FullGroupCodes
	fullGroupCodes := make([]string, 0)
	if updateResp.Diff != nil {
		fullGroupCodes = updateResp.Diff.GetAddFullGroupCodes()
		logger.Infof(ctx, "[Server] 获取新增函数组代码 - Count: %d, FullGroupCodes: %v", len(fullGroupCodes), fullGroupCodes)
	}

	// 7. 发送回调消息给 agent-server
	if len(fullGroupCodes) > 0 {
		callbackData := &dto.FunctionGenCallback{
			RecordID:       result.RecordID,
			MessageID:      result.MessageID,
			Success:        true,
			FullGroupCodes: fullGroupCodes,
			AppID:          serviceTree.App.ID,
			AppCode:        serviceTree.App.Code,
			Error:          "",
		}

		// 通过 NATS 发送回调
		callbackSubject := subjects.GetAgentServerFunctionGenCallbackSubject()
		callbackJSON, err := json.Marshal(callbackData)
		if err != nil {
			logger.Errorf(ctx, "[Server] 序列化回调消息失败: error=%v", err)
			return
		}

		callbackMsg := nats.NewMsg(callbackSubject)
		callbackMsg.Data = callbackJSON
		if traceID != "" {
			callbackMsg.Header.Set("X-Trace-Id", traceID)
		}
		if requestUser != "" {
			callbackMsg.Header.Set("X-Request-User", requestUser)
		}

		if err := s.natsConn.PublishMsg(callbackMsg); err != nil {
			logger.Errorf(ctx, "[Server] 发送回调消息失败: error=%v", err)
			// 不中断流程，记录日志即可
		} else {
			logger.Infof(ctx, "[Server] 回调消息已发送 - RecordID: %d, MessageID: %d, FullGroupCodes: %v, AppCode: %s",
				result.RecordID, result.MessageID, fullGroupCodes, serviceTree.App.Code)
		}
	} else {
		logger.Warnf(ctx, "[Server] 没有新增的函数组代码，跳过回调 - RecordID: %d", result.RecordID)
	}
}

// extractCodeFromMarkdown 从 Markdown 代码块中提取代码
// 支持格式：
// 1. ```go\n代码\n```
// 2. ```\n代码\n```
// 3. 如果找不到代码块，返回原始内容
func (s *Server) extractCodeFromMarkdown(content string) string {
	// 查找 ```go 或 ``` 开头的代码块
	lines := strings.Split(content, "\n")

	var codeBlocks []string
	var inCodeBlock bool
	var codeBlockStart int

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// 检查是否是代码块开始标记
		if strings.HasPrefix(trimmed, "```") {
			if inCodeBlock {
				// 代码块结束，提取内容
				if i > codeBlockStart {
					codeBlock := strings.Join(lines[codeBlockStart+1:i], "\n")
					codeBlocks = append(codeBlocks, codeBlock)
				}
				inCodeBlock = false
			} else {
				// 代码块开始
				inCodeBlock = true
				codeBlockStart = i
			}
			continue
		}
	}

	// 如果代码块没有正确关闭，也提取已收集的内容
	if inCodeBlock && codeBlockStart < len(lines)-1 {
		codeBlock := strings.Join(lines[codeBlockStart+1:], "\n")
		codeBlocks = append(codeBlocks, codeBlock)
	}

	// 如果有代码块，返回第一个（通常只有一个）
	if len(codeBlocks) > 0 {
		extracted := strings.TrimSpace(codeBlocks[0])
		// 如果提取的代码不为空，返回它
		if extracted != "" {
			return extracted
		}
	}

	// 如果没有找到代码块或代码块为空，返回原始内容（作为 fallback）
	return content
}

// extractGroupCodeFromCode 从生成的代码中提取 group_code
// 提取优先级：
// 1. 从 RouterGroup 定义中的 GroupCode 字段提取：GroupCode: "crm_ticket"
// 2. 从注释中提取文件名：//<文件名>crm_ticket.go</文件名>
// 3. 从结构体名称推断：type CrmTicket struct -> crm_ticket
func (s *Server) extractGroupCodeFromCode(code string) string {
	lines := strings.Split(code, "\n")

	// 1. 优先从 RouterGroup 定义中的 GroupCode 字段提取
	// 格式：GroupCode: "crm_ticket", 或 GroupCode: "crm_ticket" //注释
	for i, line := range lines {
		line = strings.TrimSpace(line)
		// 查找包含 GroupCode: 的行
		if strings.Contains(line, "GroupCode:") {
			// 提取引号中的值
			start := strings.Index(line, `"`)
			if start != -1 {
				end := strings.Index(line[start+1:], `"`)
				if end != -1 {
					groupCode := strings.TrimSpace(line[start+1 : start+1+end])
					if groupCode != "" {
						return groupCode
					}
				}
			}
			// 如果没找到引号，尝试查找单引号
			start = strings.Index(line, `'`)
			if start != -1 {
				end := strings.Index(line[start+1:], `'`)
				if end != -1 {
					groupCode := strings.TrimSpace(line[start+1 : start+1+end])
					if groupCode != "" {
						return groupCode
					}
				}
			}
			// 如果当前行没有找到，检查下一行（可能是多行定义）
			if i+1 < len(lines) {
				nextLine := strings.TrimSpace(lines[i+1])
				start := strings.Index(nextLine, `"`)
				if start != -1 {
					end := strings.Index(nextLine[start+1:], `"`)
					if end != -1 {
						groupCode := strings.TrimSpace(nextLine[start+1 : start+1+end])
						if groupCode != "" {
							return groupCode
						}
					}
				}
			}
		}
	}

	// 2. 从注释中提取文件名
	// 格式：//<文件名>crm_ticket.go</文件名>
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "<文件名>") && strings.Contains(line, "</文件名>") {
			// 提取文件名
			start := strings.Index(line, "<文件名>") + len("<文件名>")
			end := strings.Index(line, "</文件名>")
			if start < end {
				fileName := strings.TrimSpace(line[start:end])
				// 去掉 .go 后缀
				if strings.HasSuffix(fileName, ".go") {
					fileName = fileName[:len(fileName)-3]
				}
				return fileName
			}
		}
	}

	// 3. 从结构体名称推断
	// 查找第一个 type 定义，例如：type CrmTicket struct
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "type ") && strings.Contains(line, " struct") {
			// 提取结构体名称，例如：type CrmTicket struct -> CrmTicket
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				structName := parts[1]
				// 将大驼峰转换为小写下划线：CrmTicket -> crm_ticket
				groupCode := s.camelToSnake(structName)
				return groupCode
			}
		}
	}

	// 如果都找不到，返回空字符串（由调用方处理 fallback）
	return ""
}

// camelToSnake 将大驼峰转换为小写下划线
// 例如：CrmTicket -> crm_ticket, ToolsCashier -> tools_cashier
func (s *Server) camelToSnake(camel string) string {
	if camel == "" {
		return ""
	}

	var result strings.Builder
	for i, r := range camel {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteByte('_')
		}
		result.WriteRune(r)
	}

	return strings.ToLower(result.String())
}
