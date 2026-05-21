package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	_ "net/http/pprof" // 导入 pprof
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/pkg/natsx"
	"github.com/kageos/kageos/pkg/netprobe"
	"github.com/kageos/kageos/pkg/subjects"
	"github.com/kageos/kageos/sdk/agent-app/env"
	"github.com/nats-io/nats.go"
)

var (
	app                *App
	initOnce           sync.Once
	initErr            error // 记录初始化错误
	natsURLResolveOnce sync.Once
	natsURLResolved    string
	natsURLResolveErr  error
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
	transport *AppTransport
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
		return fmt.Errorf("路由 %s 已存在，不允许重复注册。已存在的路由信息: Router=%s, Method=%s",
			router, existing.Router, existing.Method)
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

// Subjects NATS 主题
type Subjects struct {
	InvokeCommand    string // app.v1.cmd.invoke.{user}.{app}.{version} - runtime 转发到 app 的调用命令
	InvokeReply      string // app-server.v1.reply.app.invoke.{user}.{app}.{version} - app 回给 app-server 的回复
	ControlCommand   string // app.v1.cmd.control.{user}.{app}.{version} - shutdown / onAppUpdate 控制命令
	LifecycleEvent   string // runtime.v1.event.lifecycle.{user}.{app}.{version} - startup / close / discovery 事件
	DiscoveryRequest string // app.v1.cmd.discovery.request - runtime 广播发现命令
}

// NewApp 创建新的应用实例
func NewApp() (*App, error) {
	if err := initAppLogger(); err != nil {
		return nil, err
	}

	conn, err := connectAppNATS()
	if err != nil {
		return nil, err
	}

	newApp := newAppInstance(conn)
	if err := initializeAppRuntime(newApp); err != nil {
		return nil, err
	}

	startPprofServer()

	logger.Infof(context.Background(), "NewApp() completed successfully")
	return newApp, nil
}

func initAppLogger() error {
	cfg := logger.Config{
		Level:      "info",
		Filename:   fmt.Sprintf("/app/workplace/logs/%s_%s_%s.log", env.User, env.App, env.Version),
		MaxSize:    100,
		MaxBackups: 3,
		MaxAge:     7,
		Compress:   true,
		IsDev:      false,
	}
	return logger.Init(cfg)
}

func connectAppNATS() (*nats.Conn, error) {
	natsURL, err := resolveAppNATSURLOnce()
	if err != nil {
		logger.Errorf(context.Background(), "Failed to resolve NATS endpoint: %v", err)
		return nil, err
	}

	name := fmt.Sprintf("agent-app-%s-%s-%s", env.User, env.App, env.Version)
	options := []nats.Option{
		nats.ErrorHandler(func(nc *nats.Conn, sub *nats.Subscription, err error) {
			logger.Errorf(context.Background(), "NATS error: %v", err)
		}),
	}
	return connectAppNATSCandidate(natsURL, name, options...)
}

func resolveAppNATSURLOnce() (string, error) {
	natsURLResolveOnce.Do(func() {
		natsURLResolved, natsURLResolveErr = probeAppNATSURL(resolveNATSURL())
	})
	return natsURLResolved, natsURLResolveErr
}

func probeAppNATSURL(natsURL string) (string, error) {
	candidates := netprobe.URLCandidates(natsURL)
	if len(candidates) <= 1 {
		return natsURL, nil
	}

	var failures []string
	for _, candidate := range candidates {
		err := probeNATSStatus(candidate, time.Second)
		if err == nil {
			if candidate != natsURL {
				logger.Infof(context.Background(), "NATS endpoint auto-resolved: %s -> %s", natsURL, candidate)
			}
			return candidate, nil
		}
		failures = append(failures, fmt.Sprintf("%s: %v", candidate, err))
	}

	err := fmt.Errorf("failed to resolve NATS candidates: %s", strings.Join(failures, "; "))
	logger.Errorf(context.Background(), "%v", err)
	return natsURL, err
}

func probeNATSStatus(natsURL string, timeout time.Duration) error {
	conn, err := nats.Connect(
		natsURL,
		nats.Name("agent-app-nats-probe"),
		nats.Timeout(timeout),
		nats.NoReconnect(),
	)
	if err != nil {
		return err
	}
	defer conn.Close()
	if conn.Status() != nats.CONNECTED {
		return fmt.Errorf("unexpected NATS status: %v", conn.Status())
	}
	return nil
}

func connectAppNATSCandidate(natsURL, name string, options ...nats.Option) (*nats.Conn, error) {
	logger.Infof(context.Background(), "Connecting to NATS: %s", natsURL)
	conn, err := natsx.ConnectNamedWithOptions(natsURL, name, options...)
	if err != nil {
		logger.Errorf(context.Background(), "Failed to connect to NATS %s: %v", natsURL, err)
		return nil, fmt.Errorf("failed to connect to NATS %s: %w", natsURL, err)
	}
	logger.Infof(context.Background(), "NATS connected successfully to %s", conn.ConnectedUrl())
	return conn, nil
}

func resolveNATSURL() string {
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://127.0.0.1:4222"
	}
	return natsURL
}

func newAppInstance(conn *nats.Conn) *App {
	newApp := &App{
		Context:         context.Background(),
		exit:            make(chan struct{}),
		conn:            conn,
		startTime:       time.Now(), // 记录启动时间
		routerInfo:      make(map[string]*routerInfo),
		packageContexts: make(map[string]*PackageContext),
		subjects:        buildAppSubjects(),
	}
	newApp.transport = NewAppTransport(newApp.conn, newApp.subjects)
	return newApp
}

func buildAppSubjects() *Subjects {
	return &Subjects{
		InvokeCommand:    subjects.BuildAppInvokeSubject(env.User, env.App, env.Version),
		InvokeReply:      subjects.BuildAppServerAppInvokeReplySubject(env.User, env.App, env.Version),
		ControlCommand:   subjects.BuildAppControlSubject(env.User, env.App, env.Version),
		LifecycleEvent:   subjects.BuildRuntimeLifecycleEventSubject(env.User, env.App, env.Version),
		DiscoveryRequest: subjects.AppDiscoveryRequestSubject,
	}
}

func initializeAppRuntime(newApp *App) error {
	logger.Infof(context.Background(), "Initializing router...")
	initRouter(newApp)
	logger.Infof(context.Background(), "Router initialized")

	// 注册 NATS 订阅（subject 硬编码在 nats_router.go，方便阅读）
	if err := registerNATS(newApp); err != nil {
		return err
	}
	return nil
}

func startPprofServer() {
	// 启动 pprof HTTP 服务器（用于性能分析）
	// 监听在 6060 端口，可以通过 http://localhost:6060/debug/pprof/ 访问
	go func() {
		pprofAddr := ":6060"
		logger.Infof(context.Background(), "Starting pprof server on %s", pprofAddr)
		if err := http.ListenAndServe(pprofAddr, nil); err != nil {
			logger.Warnf(context.Background(), "pprof server failed: %v", err)
		}
	}()
}

func (a *App) notifyStartupBestEffort() {
	// 发送启动完成通知给 runtime
	// 通知 runtime 新版本已经成功启动并准备好接收请求
	logger.Infof(context.Background(), "Sending startup notification...")
	if err := a.sendStartupNotification(); err != nil {
		logger.Warnf(context.Background(), "Failed to send startup notification: %v", err)
		// 不返回错误，启动通知失败不应阻止应用运行
	} else {
		logger.Infof(context.Background(), "Startup notification sent successfully")
	}
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
	if err := a.transport.PublishInvokeResponse(resp, true); err != nil {
		logger.Errorf(context.Background(), "Failed to publish invoke response: %v", err)
	}
}

func (a *App) sendErrResponse(resp *dto.RequestAppResp) {
	if err := a.transport.PublishInvokeResponse(resp, false); err != nil {
		logger.Errorf(context.Background(), "Failed to publish invoke error response: %v", err)
	}
}

// handleDiscovery 处理发现消息
func (a *App) handleDiscovery(msg *nats.Msg) {
	discoveryMsg, err := a.transport.ParseDiscoveryRequest(msg)
	if err != nil {
		logger.Errorf(context.Background(), "Failed to parse discovery request: %v", err)
		return
	}
	if err := a.transport.PublishDiscoveryResponse(discoveryMsg.RuntimeID, a.startTime); err != nil {
		logger.Errorf(context.Background(), "Failed to publish discovery response: %v", err)
		return
	}
}

// Close 关闭应用
func (a *App) Close() error {
	logger.Infof(context.Background(), "App.Close() called")

	if !a.markShutdownRequested() {
		return nil
	}

	a.notifyCloseBestEffort()
	a.unsubscribeAll()
	a.closeNATSConnection()
	a.cleanupRuntimeResources()
	a.closeExitSignal()

	logger.Infof(context.Background(), "App.Close() completed, all resources released")

	return nil
}

func (a *App) markShutdownRequested() bool {
	a.shutdownMu.Lock()
	defer a.shutdownMu.Unlock()

	if a.shutdownRequested {
		logger.Infof(context.Background(), "Shutdown already in progress, skipping cleanup")
		return false
	}

	a.shutdownRequested = true
	return true
}

func (a *App) notifyCloseBestEffort() {
	if err := a.sendCloseNotification(); err != nil {
		logger.Warnf(context.Background(), "Failed to send close notification: %v", err)
		// 不返回错误，通知失败不应阻止关闭流程
	}
}

func (a *App) unsubscribeAll() {
	for _, sub := range a.subs {
		sub.Unsubscribe()
	}
}

func (a *App) closeNATSConnection() {
	if a.conn != nil {
		a.conn.Close()
	}
}

func (a *App) cleanupRuntimeResources() {
	closeAllDatabases()
	GetFileCache().CleanupOnShutdown()
	forceGCAndFreeMemory()
}

func (a *App) closeExitSignal() {
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

	if err := app.CompileAndValidate(); err != nil {
		logger.Errorf(context.Background(), "App schema compile failed: %v", err)
		if notifyErr := app.transport.PublishStartupFailure(app.startTime, err.Error()); notifyErr != nil {
			logger.Errorf(context.Background(), "Failed to publish startup failure: %v", notifyErr)
		}
		return err
	}

	app.notifyStartupBestEffort()

	err := app.Start(context.Background())
	if err != nil {
		logger.Errorf(context.Background(), "App.Start() failed: %v", err)
		return err
	}
	return nil
}

// sendStartupNotification 发送启动完成通知
func (a *App) sendStartupNotification() error {
	return a.transport.PublishStartup(a.startTime)
}

// sendCloseNotification 发送应用关闭通知
func (a *App) sendCloseNotification() error {
	return a.transport.PublishClose(a.startTime, time.Now())
}

// handleShutdownCommand 处理 runtime 发送的关闭命令
func (a *App) handleShutdownCommand(message subjects.Message) {
	ctx := context.Background()
	logger.Infof(ctx, "Received shutdown command from runtime: %s/%s/%s", message.User, message.App, message.Version)

	if !a.markRuntimeShutdownRequested(ctx) {
		return
	}

	a.waitForRuntimeShutdownDrain(ctx)
	a.resetShutdownRequestedForCleanup()
	a.closeAfterRuntimeShutdown(ctx)

	logger.Infof(ctx, "Application shutdown initiated by runtime command")
}

func (a *App) markRuntimeShutdownRequested(ctx context.Context) bool {
	a.shutdownMu.Lock()
	defer a.shutdownMu.Unlock()

	if a.shutdownRequested {
		logger.Infof(ctx, "Shutdown already in progress, ignoring duplicate shutdown command")
		return false
	}

	a.shutdownRequested = true
	return true
}

func (a *App) waitForRuntimeShutdownDrain(ctx context.Context) {
	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := a.waitForAllFunctionsToComplete(shutdownCtx, 30*time.Second); err != nil {
		logger.Warnf(ctx, "Some functions did not complete in time: %v", err)
	} else {
		logger.Infof(ctx, "All functions completed successfully")
	}
}

func (a *App) resetShutdownRequestedForCleanup() {
	a.shutdownMu.Lock()
	a.shutdownRequested = false // 临时重置，让 Close() 执行清理
	a.shutdownMu.Unlock()
}

func (a *App) closeAfterRuntimeShutdown(ctx context.Context) {
	if err := a.Close(); err != nil {
		logger.Warnf(ctx, "Error during Close(): %v", err)
	}
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

// handleAppControlMessage 处理 App 控制消息（shutdown、onAppUpdate）。
func (a *App) handleAppControlMessage(msg *nats.Msg) {
	var message subjects.Message
	if err := json.Unmarshal(msg.Data, &message); err != nil {
		logger.Errorf(context.Background(), "Failed to unmarshal app control message: %v", err)
		return
	}
	logger.Infof(context.Background(), "Received app control message: %+v", message)

	switch message.Type {
	case subjects.MessageTypeStatusShutdown:
		a.handleShutdownCommand(message)
	case subjects.MessageTypeStatusDiscovery:
		a.handleDiscovery(msg) // 发现消息还是用原来的格式
	case subjects.MessageTypeStatusOnAppUpdate:
		a.onAppUpdate(msg) // 发现消息还是用原来的格式
	default:
		logger.Warnf(context.Background(), "Unknown app control message type: %s", message.Type)
	}
}

// handleRuntimeStatusMessage 方法已移除。
// runtime.v1.event.lifecycle.*.*.* 是应用发送给 Runtime 的事件主题，不需要在应用内订阅。

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
