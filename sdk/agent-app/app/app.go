package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	_ "net/http/pprof" // 导入 pprof
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/discovery"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/pkg/subjects"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/env"
	"github.com/nats-io/nats.go"
)

var (
	app      *App
	initOnce sync.Once
	initErr  error // 记录初始化错误
)

func initApp() {
	initOnce.Do(func() {
		var err error
		app, err = NewApp()
		if err != nil {
			initErr = err
			fmt.Printf("Failed to init app: %v\n", err)
			// 不要 panic，让错误在 Run() 时处理
			// 如果是在 init() 中调用，panic 会导致应用无法启动
			return
		}
		//POST("/test/add", AddHandle, Temp)
		//POST("/test/get", GetHandle, Temp)
	})
}

// App SDK 应用基类
type App struct {
	conn      *nats.Conn
	subjects  *Subjects
	subs      []*nats.Subscription
	exit      chan struct{}
	startTime time.Time // 应用启动时间

	routerInfo      map[string]*routerInfo
	packageContexts map[string]*PackageContext

	context.Context
	// 运行中函数的计数
	runningCount      int32
	shutdownRequested bool
	shutdownMu        sync.RWMutex

	//fileCache
}

// HTTP 方法常量已移除，直接使用字符串 "POST", "GET", "PUT", "DELETE" 即可

func (a *App) registerRouter(method string, router string, handler HandleFunc, templater Templater) {
	// 系统路由（如 /_callback）没有 package 路径，传递 nil options
	// 通过 PackageContext 注册的路由会有 PackagePath
	// 使用统一的 addRoute 方法
	if err := a.addRoute(router, method, handler, templater, nil); err != nil {
		logger.Errorf(context.Background(), "Failed to register router %s %s: %v", method, router, err)
		panic(err) // 注册失败时 panic，避免静默失败
	}
}

// addRoute 添加路由（统一设置路由的方法）
// 检查 URL 唯一性，如果已存在则返回错误
func (a *App) addRoute(router string, method string, handleFunc HandleFunc, templater Templater, options *RegisterOptions) error {
	key := routerKey(router)

	// 检查 URL 唯一性
	if existing, exists := a.routerInfo[key]; exists {
		logger.Errorf(context.Background(), fmt.Sprintf("路由 %s 已存在，不允许重复注册。已存在的路由信息: Router=%s, Method=%s",
			router, existing.Router, existing.Method))
	}

	a.routerInfo[key] = &routerInfo{
		HandleFunc: handleFunc,
		Router:     router,
		Method:     method,
		Options:    options,
		Template:   templater,
	}
	return nil
}

// getRoute 获取路由（统一获取路由的方法）
// router: 路由路径（不包含 method）
func (a *App) getRoute(router string) (*routerInfo, error) {
	key := routerKey(router)
	info, ok := a.routerInfo[key]
	if ok {
		return info, nil
	}

	logger.Infof(a, "Router %s not found routerInfo:%+v ", key, a.routerInfo)
	return nil, fmt.Errorf("router %s not found", router)
}

// Subjects NATS 主题（重构后）
type Subjects struct {
	// 保持独立的复杂主题
	AppRequest  string // app_runtime.app.{user}.{app}.{version} - 应用请求
	AppResponse string // app.function_server.{user}.{app}.{version} - 应用响应

	// 简化的状态通知主题
	AppStatus     string // app.status.{user}.{app}.{version} - 处理 shutdown、discovery、onAppUpdate
	RuntimeStatus string // runtime.status.{user}.{app}.{version} - 处理 startup、close、discovery
	Discovery     string // ai-agent-os.runtime.discovery 处理服务发现
}

// NewApp 创建新的应用实例
func NewApp() (*App, error) {

	cfg := logger.Config{
		Level:      "info",
		Filename:   fmt.Sprintf("/app/workplace/logs/%s_%s_%s.log", env.User, env.App, env.Version),
		MaxSize:    100,
		MaxBackups: 3,
		MaxAge:     7,
		Compress:   true,
		IsDev:      false,
	}
	err := logger.Init(cfg)
	if err != nil {
		return nil, err
	}

	// 连接 NATS（优先使用环境变量）
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://127.0.0.1:4222"
	}

	logger.Infof(context.Background(), "Connecting to NATS: %s", natsURL)

	// 设置连接选项，包括超时
	opts := []nats.Option{
		nats.Timeout(10 * time.Second),      // 连接超时 10 秒
		nats.ReconnectWait(2 * time.Second), // 重连等待时间
		nats.MaxReconnects(5),               // 最大重连次数
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				logger.Warnf(context.Background(), "NATS disconnected: %v", err)
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Infof(context.Background(), "NATS reconnected to %s", nc.ConnectedUrl())
		}),
		nats.ErrorHandler(func(nc *nats.Conn, sub *nats.Subscription, err error) {
			logger.Errorf(context.Background(), "NATS error: %v", err)
		}),
	}

	conn, err := nats.Connect(natsURL, opts...)
	if err != nil {
		logger.Errorf(context.Background(), "Failed to connect to NATS: %v", err)
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	logger.Infof(context.Background(), "NATS connected successfully to %s", conn.ConnectedUrl())

	newApp := &App{
		Context:         context.Background(),
		exit:            make(chan struct{}),
		conn:            conn,
		startTime:       time.Now(), // 记录启动时间
		routerInfo:      make(map[string]*routerInfo),
		packageContexts: make(map[string]*PackageContext),
		subjects: &Subjects{
			// 保持独立的复杂主题
			AppRequest:  subjects.BuildAppRuntime2AppSubject(env.User, env.App, env.Version),
			AppResponse: subjects.BuildApp2FunctionServerSubject(env.User, env.App, env.Version),

			// 简化的状态通知主题（AppStatus 也用于 app-runtime 的 update 回调请求）
			AppStatus:     subjects.BuildAppStatusSubject(env.User, env.App, env.Version),
			RuntimeStatus: subjects.BuildRuntimeStatusSubject(env.User, env.App, env.Version),
			Discovery:     subjects.GetRuntimeDiscoverySubject(),
		},
	}

	logger.Infof(context.Background(), "Initializing router...")
	initRouter(newApp)
	logger.Infof(context.Background(), "Router initialized")

	// 注册 NATS 订阅（subject 硬编码在 nats_router.go，方便阅读）
	if err := registerNATS(newApp); err != nil {
		return nil, err
	}

	// 启动 pprof HTTP 服务器（用于性能分析）
	// 监听在 6060 端口，可以通过 http://localhost:6060/debug/pprof/ 访问
	go func() {
		pprofAddr := ":6060"
		logger.Infof(context.Background(), "Starting pprof server on %s", pprofAddr)
		if err := http.ListenAndServe(pprofAddr, nil); err != nil {
			logger.Warnf(context.Background(), "pprof server failed: %v", err)
		}
	}()

	// 发送启动完成通知给 runtime
	// 通知 runtime 新版本已经成功启动并准备好接收请求
	logger.Infof(context.Background(), "Sending startup notification...")
	if err := newApp.sendStartupNotification(); err != nil {
		logger.Warnf(context.Background(), "Failed to send startup notification: %v", err)
		// 不返回错误，启动通知失败不应阻止应用运行
	} else {
		logger.Infof(context.Background(), "Startup notification sent successfully")
	}

	logger.Infof(context.Background(), "NewApp() completed successfully")
	return newApp, nil
}

// Start 启动应用
func (a *App) Start(ctx context.Context) error {
	fmt.Printf("starting app %s %s %s \n", env.User, env.App, env.Version)
	logger.Infof(ctx, "App started successfully, waiting for messages...")

	// 添加 panic 恢复机制
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf(ctx, "App panic recovered: %v", r)
		}
	}()

	// 保持连接
	select {
	case <-ctx.Done():
		logger.Infof(ctx, "Context cancelled, shutting down...")
		// 不要在这里发送关闭通知，因为连接可能已被Close()方法关闭
		// 关闭通知会在Close()方法中发送，确保正确的关闭顺序
		return ctx.Err()
	case <-a.exit:
		logger.Infof(ctx, "Exit signal received, shutting down...")
		// 不要在这里发送关闭通知，因为连接可能已被Close()方法关闭
		// 关闭通知会在Close()方法中发送，确保正确的关闭顺序
		return nil
	}
}

// sendResponse 发送响应消息
func (a *App) sendResponse(resp *dto.RequestAppResp) {
	data, err := json.Marshal(resp)
	if err != nil {
		return
	}

	// 创建带 header 的消息
	msg := &nats.Msg{
		Subject: a.subjects.AppResponse,
		Data:    data,
		Header:  make(nats.Header),
	}
	logger.Infof(context.Background(), "Sending response: %s", msg.Subject)

	// 设置 trace_id header
	if resp.TraceId != "" {
		msg.Header.Set(contextx.TraceIdHeader, resp.TraceId)
	}
	msg.Header.Set("code", "0")

	if err := a.conn.PublishMsg(msg); err != nil {
		return
	}
}

// PublishMessage 将消息发送到 NATS 主题，由消息服务消费（渠道由消费方决定）
func (a *App) PublishMessage(payload *dto.MessageSendPayload) error {
	if a.conn == nil {
		return fmt.Errorf("NATS connection is nil")
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal message payload: %w", err)
	}
	subject := subjects.GetMessageSendSubject()
	if err := a.conn.Publish(subject, data); err != nil {
		return fmt.Errorf("publish message to %s: %w", subject, err)
	}
	logger.Infof(context.Background(), "[PublishMessage] published to %s, to_users=%s title=%s", subject, payload.ToUsers, payload.Title)
	return nil
}

func (a *App) sendErrResponse(resp *dto.RequestAppResp) {
	data, err := json.Marshal(resp)
	if err != nil {
		logger.Errorf(context.Background(), "Failed to marshal response: %v", err)
		return
	}

	// 创建带 header 的消息
	msg := &nats.Msg{
		Subject: a.subjects.AppResponse,
		Data:    data,
		Header:  make(nats.Header),
	}

	// 设置 trace_id header
	if resp.TraceId != "" {
		msg.Header.Set(contextx.TraceIdHeader, resp.TraceId)
	}
	msg.Header.Set("code", "-1")
	msg.Header.Set("msg", resp.Error)

	if err := a.conn.PublishMsg(msg); err != nil {
		return
	}
}

// handleDiscovery 处理发现消息
func (a *App) handleDiscovery(msg *nats.Msg) {
	var discoveryMsg discovery.DiscoveryMessage
	if err := json.Unmarshal(msg.Data, &discoveryMsg); err != nil {
		return
	}

	// 构建发现响应数据
	responseData := map[string]interface{}{
		"type":       "response",
		"status":     "running",
		"runtime_id": discoveryMsg.RuntimeID,
		"start_time": a.startTime,
		"timestamp":  time.Now(),
	}

	// 使用新的统一消息格式
	message := subjects.Message{
		Type:      subjects.MessageTypeStatusDiscovery,
		User:      env.User,
		App:       env.App,
		Version:   env.Version,
		Data:      responseData,
		Timestamp: time.Now(),
	}

	messageData, err := json.Marshal(message)
	if err != nil {
		logger.Errorf(context.Background(), "Failed to marshal discovery response: %v", err)
		return
	}

	// 发送到 Runtime 状态主题
	subject := subjects.BuildRuntimeStatusSubject(env.User, env.App, env.Version)
	if err := a.conn.Publish(subject, messageData); err != nil {
		logger.Errorf(context.Background(), "Failed to publish discovery response: %v", err)
		return
	}

	logger.Infof(context.Background(), "Discovery response sent to subject: %s", subject)
}

// Close 关闭应用
func (a *App) Close() error {
	logger.Infof(context.Background(), "App.Close() called")

	// 检查是否已经关闭过（使用原子操作避免重复清理）
	a.shutdownMu.Lock()
	alreadyClosed := a.shutdownRequested
	if alreadyClosed {
		// 如果已经在关闭过程中，避免重复关闭
		logger.Infof(context.Background(), "Shutdown already in progress, skipping cleanup")
		a.shutdownMu.Unlock()
		return nil
	}
	// 设置关闭请求标志
	a.shutdownRequested = true
	a.shutdownMu.Unlock()

	// 1. 先发送关闭通知（在连接关闭前）
	if err := a.sendCloseNotification(); err != nil {
		logger.Warnf(context.Background(), "Failed to send close notification: %v", err)
		// 不返回错误，通知失败不应阻止关闭流程
	}

	// 2. 取消所有订阅
	for _, sub := range a.subs {
		sub.Unsubscribe()
	}

	// 3. 关闭NATS连接
	if a.conn != nil {
		a.conn.Close()
	}

	// 4. 关闭所有数据库连接
	closeAllDatabases()

	// 5. 清理文件缓存（立即删除所有无引用的待删除文件）
	GetFileCache().CleanupOnShutdown()

	// 6. 强制 GC 并释放内存回操作系统
	forceGCAndFreeMemory()

	// 7. 安全地关闭退出channel
	a.shutdownMu.Lock()
	defer a.shutdownMu.Unlock()

	select {
	case <-a.exit:
		// channel已经关闭，避免重复关闭
		logger.Infof(context.Background(), "Exit channel already closed")
	default:
		// channel未关闭，安全关闭
		close(a.exit)
		logger.Infof(context.Background(), "Exit channel closed")
	}

	logger.Infof(context.Background(), "App.Close() completed, all resources released")

	return nil
}

func Run() error {
	if app == nil {
		initApp()
	}

	// 检查初始化错误
	if initErr != nil {
		return fmt.Errorf("app initialization failed: %w", initErr)
	}

	if app == nil {
		return fmt.Errorf("app is nil after initialization")
	}

	// 确保在 Start() 返回后调用 Close() 清理资源
	// 无论是正常退出还是异常退出，都要清理资源
	// 注意：Close() 已经在 handleShutdownCommand 中调用过了，这里只是兜底
	defer func() {
		if app != nil {
			app.Close()
		}
	}()

	err := app.Start(context.Background())
	if err != nil {
		logger.Errorf(context.Background(), "App.Start() failed: %v", err)
		return err
	}
	return nil
}

// sendStartupNotification 发送启动完成通知
func (a *App) sendStartupNotification() error {
	// 构建启动通知消息（只包含业务数据，不重复标识信息）
	notification := map[string]interface{}{
		"status":     "started",
		"start_time": a.startTime,
	}

	// 使用新的消息格式
	message := subjects.Message{
		Type:      subjects.MessageTypeStatusStartup,
		User:      env.User,
		App:       env.App,
		Version:   env.Version,
		Data:      notification,
		Timestamp: time.Now(),
	}

	messageData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal startup message: %w", err)
	}

	// 发送到 Runtime 状态主题
	subject := subjects.BuildRuntimeStatusSubject(env.User, env.App, env.Version)
	if err := a.conn.Publish(subject, messageData); err != nil {
		return fmt.Errorf("failed to publish startup notification: %w", err)
	}

	logger.Infof(context.Background(), "Startup notification sent to subject: %s", subject)
	return nil
}

// sendCloseNotification 发送应用关闭通知
func (a *App) sendCloseNotification() error {
	// 构建关闭通知消息（只包含业务数据，不重复标识信息）
	notification := map[string]interface{}{
		"status":     "closed",
		"start_time": a.startTime,
		"close_time": time.Now(),
	}

	// 使用新的消息格式
	message := subjects.Message{
		Type:      subjects.MessageTypeStatusClose,
		User:      env.User,
		App:       env.App,
		Version:   env.Version,
		Data:      notification,
		Timestamp: time.Now(),
	}

	messageData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal close message: %w", err)
	}

	// 发送到 Runtime 状态主题
	subject := subjects.BuildRuntimeStatusSubject(env.User, env.App, env.Version)
	if err := a.conn.Publish(subject, messageData); err != nil {
		return fmt.Errorf("failed to publish close notification: %w", err)
	}

	logger.Infof(context.Background(), "Close notification sent to subject: %s", subject)
	return nil
}

// handleShutdownCommand 处理 runtime 发送的关闭命令
func (a *App) handleShutdownCommand(message subjects.Message) {
	ctx := context.Background()
	logger.Infof(ctx, "Received shutdown command from runtime: %s/%s/%s", message.User, message.App, message.Version)

	// 检查是否已经在关闭过程中
	a.shutdownMu.Lock()
	if a.shutdownRequested {
		// 如果已经在关闭过程中，忽略重复的关闭命令
		logger.Infof(ctx, "Shutdown already in progress, ignoring duplicate shutdown command")
		a.shutdownMu.Unlock()
		return
	}
	// 先设置标志，防止并发关闭
	a.shutdownRequested = true
	a.shutdownMu.Unlock()

	// 等待所有运行中的函数完成
	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := a.waitForAllFunctionsToComplete(shutdownCtx, 30*time.Second); err != nil {
		logger.Warnf(ctx, "Some functions did not complete in time: %v", err)
	} else {
		logger.Infof(ctx, "All functions completed successfully")
	}

	// 调用 Close() 方法清理所有资源（NATS连接、订阅、文件缓存、强制GC等）
	// Close() 会检查 shutdownRequested，如果已经设置会跳过清理
	// 但我们已经设置了，所以需要临时重置让 Close() 执行清理
	a.shutdownMu.Lock()
	a.shutdownRequested = false // 临时重置，让 Close() 执行清理
	a.shutdownMu.Unlock()

	if err := a.Close(); err != nil {
		logger.Warnf(ctx, "Error during Close(): %v", err)
	}

	logger.Infof(ctx, "Application shutdown initiated by runtime command")
}

// incrementRunningCount 增加运行中函数计数
func (a *App) incrementRunningCount() {
	atomic.AddInt32(&a.runningCount, 1)
}

// decrementRunningCount 减少运行中函数计数
func (a *App) decrementRunningCount() {
	atomic.AddInt32(&a.runningCount, -1)
}

// getRunningCount 获取运行中函数的数量
func (a *App) getRunningCount() int32 {
	return atomic.LoadInt32(&a.runningCount)
}

// waitForAllFunctionsToComplete 等待所有函数完成
func (a *App) waitForAllFunctionsToComplete(ctx context.Context, timeout time.Duration) error {
	logger.Infof(ctx, "Waiting for all functions to complete...")

	start := time.Now()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			count := a.getRunningCount()
			if count == 0 {
				logger.Infof(ctx, "All functions completed in %v", time.Since(start))
				return nil
			}

			if time.Since(start) > timeout {
				logger.Warnf(ctx, "Timeout waiting for functions to complete, %d still running", count)
				return fmt.Errorf("timeout waiting for %d functions to complete", count)
			}

			logger.Infof(ctx, "Still waiting for %d functions to complete...", count)
		}
	}
}

// handleAppStatusMessage 处理 App 状态消息（shutdown、discovery）
func (a *App) handleAppStatusMessage(msg *nats.Msg) {
	var message subjects.Message
	if err := json.Unmarshal(msg.Data, &message); err != nil {
		logger.Errorf(context.Background(), "Failed to unmarshal app status message: %v", err)
		return
	}
	logger.Infof(context.Background(), "Received app status message: %+v", message)

	switch message.Type {
	case subjects.MessageTypeStatusShutdown:
		a.handleShutdownCommand(message)
	case subjects.MessageTypeStatusDiscovery:
		a.handleDiscovery(msg) // 发现消息还是用原来的格式
	case subjects.MessageTypeStatusOnAppUpdate:
		a.onAppUpdate(msg) // 发现消息还是用原来的格式
	default:
		logger.Warnf(context.Background(), "Unknown app status message type: %s", message.Type)
	}
}

//// handleUpdateCallbackRequest 处理 Update Callback 请求（Request/Reply 模式）
//func (a *App) handleUpdateCallbackRequest(msg *nats.ErrorMsg) {
//
//	var request subjects.Message
//	if err := json.Unmarshal(msg.Data, &request); err != nil {
//		logger.Errorf(context.Background(), "Failed to unmarshal update callback request: %v", err)
//		// 发送错误响应
//		msgx.RespFailMsg(msg, fmt.Errorf("failed to unmarshal request: %w", err))
//		return
//	}
//
//	// 验证消息类型
//	if request.Type != subjects.MessageTypeUpdateCallbackRequest {
//		logger.Warnf(context.Background(), "Invalid message type: %s (expected: %s)", request.Type, subjects.MessageTypeUpdateCallbackRequest)
//		// 发送错误响应
//		msgx.RespFailMsg(msg, fmt.Errorf("invalid message type: %s", request.Type))
//		return
//	}
//	// 处理 update 回调逻辑（复用现有的 onAppUpdate 逻辑）
//	response, err := a.processUpdateCallback(request)
//	if err != nil {
//		logger.Errorf(context.Background(), "❌ Update callback processing failed: %v", err)
//		// 发送错误响应
//		msgx.RespFailMsg(msg, err)
//		return
//	}
//	// 发送成功响应
//	if err := msgx.RespSuccessMsg(msg, response); err != nil {
//		logger.Errorf(context.Background(), "Failed to send update callback response: %v", err)
//	} else {
//		logger.Infof(context.Background(), "📤 Update callback response sent successfully")
//	}
//}

// processUpdateCallback 处理 update 回调的核心逻辑
//func (a *App) processUpdateCallback(request subjects.Message) (interface{}, error) {
//	logger.Infof(context.Background(), "🔄 Processing update callback for trigger: %+v", request.Data)
//
//	// 直接在这里实现 update 逻辑，复用 onAppUpdate 的核心逻辑
//	// 1. 获取当前所有API和表
//	currentApis, tables, err := a.getApis()
//	if err != nil {
//		return nil, fmt.Errorf("failed to get current APIs: %w", err)
//	}
//
//	logger.Infof(context.Background(), "📋 Found %d current APIs and %d tables", len(currentApis), len(tables))
//
//	// 🔥 修复：先从上一个版本文件加载API信息，用于获取正确的 AddedVersion
//	previousVersionFile := a.getPreviousVersionFile()
//	previousApis, err := a.loadVersion(previousVersionFile)
//	if err != nil {
//		logger.Warnf(context.Background(), "Failed to load previous version file: %v", err)
//		// 不返回错误，继续执行
//	} else {
//		logger.Infof(context.Background(), "Loaded %d APIs from previous version", len(previousApis))
//	}
//
//	// 创建 API 映射，用于快速查找
//	previousApiMap := make(map[string]*model.ApiInfo)
//	for _, api := range previousApis {
//		key := a.getApiKey(api)
//		previousApiMap[key] = api
//	}
//
//	// 修正当前API的 AddedVersion
//	for _, api := range currentApis {
//		key := a.getApiKey(api)
//		if previousApi, exists := previousApiMap[key]; exists {
//			// 已存在的API，保持原有的 AddedVersion
//			api.AddedVersion = previousApi.AddedVersion
//			logger.Debugf(context.Background(), "Preserved AddedVersion for %s: %s", key, api.AddedVersion)
//		}
//		// 新API保持 AddedVersion = env.Version（已在getApis中设置）
//	}
//
//	// 2. 执行数据库迁移（如果有数据库连接）
//	db := getGormDB()
//	if db != nil {
//		logger.Infof(context.Background(), "🗄️ Performing database migration for %d tables", len(tables))
//		for _, table := range tables {
//			if err := db.AutoMigrate(table); err != nil {
//				return nil, fmt.Errorf("failed to migrate table: %w", err)
//			}
//		}
//		logger.Infof(context.Background(), "✅ Database migration completed successfully")
//	}
//
//	// 3. 保存当前版本到API日志
//	if err := a.saveCurrentVersion(currentApis); err != nil {
//		return nil, fmt.Errorf("failed to save current version: %w", err)
//	}
//
//	// 4. 执行API差异对比
//	add, update, delete, err := a.diffApi()
//	if err != nil {
//		return nil, fmt.Errorf("failed to diff APIs: %w", err)
//	}
//
//	// 5. 构建差异结果
//	diffData := map[string]interface{}{
//		"added_apis":   add,
//		"updated_apis": update,
//		"deleted_apis": delete,
//	}
//
//	// 6. 构建最终响应
//	response := map[string]interface{}{
//		"status":    "success",
//		"message":   "API diff completed successfully",
//		"diff":      diffData,
//		"version":   env.Version,
//		"timestamp": time.Now(),
//		"trigger":   request.Data,
//	}
//
//	logger.Infof(context.Background(), "📊 Generated update callback response: %+v", response)
//	logger.Infof(context.Background(), "✅ API diff summary - Added: %d, Updated: %d, Deleted: %d", len(add), len(update), len(delete))
//
//	return response, nil
//}

//// createOnAppUpdateMessage 创建用于 onAppUpdate 的消息数据
//func (a *App) createOnAppUpdateMessage(data interface{}) []byte {
//	message := subjects.Message{
//		Type:      subjects.MessageTypeStatusOnAppUpdate,
//		User:      env.User,
//		App:       env.App,
//		Version:   env.Version,
//		Data:      data,
//		Timestamp: time.Now(),
//	}
//
//	messageData, err := json.Marshal(message)
//	if err != nil {
//		logger.Errorf(context.Background(), "Failed to marshal onAppUpdate message: %v", err)
//		// 返回基本的消息数据
//		message.Data = map[string]interface{}{"fallback": true}
//		messageData, _ = json.Marshal(message)
//	}
//
//	return messageData
//}

// handleRuntimeStatusMessage 方法已移除
// RuntimeStatus 主题是应用发送给 Runtime 的，不需要接收

// forceGCAndFreeMemory 强制 GC 并释放内存回操作系统
// 在应用关闭时调用，帮助减少内存占用
func forceGCAndFreeMemory() {
	// 记录清理前的内存统计
	var mBefore runtime.MemStats
	runtime.ReadMemStats(&mBefore)
	logger.Infof(context.Background(), "[Memory] Before GC: Alloc=%d KB, Sys=%d KB, NumGC=%d, HeapSys=%d KB",
		mBefore.Alloc/1024, mBefore.Sys/1024, mBefore.NumGC, mBefore.HeapSys/1024)

	// 5. 记录清理后的内存统计信息（用于调试）
	var mAfter runtime.MemStats
	runtime.ReadMemStats(&mAfter)
	logger.Infof(context.Background(), "[Memory] After GC: Alloc=%d KB, Sys=%d KB, NumGC=%d, HeapSys=%d KB, Freed=%d KB",
		mAfter.Alloc/1024, mAfter.Sys/1024, mAfter.NumGC, mAfter.HeapSys/1024, (mBefore.Alloc-mAfter.Alloc)/1024)

	// 6. 记录系统内存变化（Sys 的变化更重要）
	sysDiff := int64(mAfter.Sys) - int64(mBefore.Sys)
	if sysDiff > 0 {
		logger.Warnf(context.Background(), "[Memory] Warning: Sys increased by %d KB (Go may not release memory to OS immediately)", sysDiff/1024)
	} else {
		logger.Infof(context.Background(), "[Memory] Sys decreased by %d KB", -sysDiff/1024)
	}
}
