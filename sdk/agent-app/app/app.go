package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ai-agent-os/ai-agent-os/dto"

	"github.com/ai-agent-os/ai-agent-os/pkg/discovery"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"

	"github.com/ai-agent-os/ai-agent-os/pkg/subjects"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/env"
	"github.com/nats-io/nats.go"
)

var app *App

func initApp() {
	fmt.Println("initApp() called")
	if app != nil {
		fmt.Println("App already exists, returning")
		return
	}
	var err error
	app, err = NewApp()
	if err != nil {
		fmt.Printf("Failed to init app: %v\n", err)
		panic(err)
	}
	//POST("/test/add", AddHandle, Temp)
	//POST("/test/get", GetHandle, Temp)

	fmt.Println("initApp() completed successfully")
}

// App SDK 应用基类
type App struct {
	conn      *nats.Conn
	subjects  *Subjects
	subs      []*nats.Subscription
	exit      chan struct{}
	startTime time.Time // 应用启动时间

	routerInfo map[string]*routerInfo

	context.Context
	// 运行中函数的计数
	runningCount      int32
	shutdownRequested bool
	shutdownMu        sync.RWMutex
}

func (a *App) getRouter(router string, method string) (*routerInfo, error) {
	trim := strings.Trim(router, "/")
	key := fmt.Sprintf("%s.%s", trim, method)
	info, ok := a.routerInfo[key]
	if ok {
		return info, nil
	}

	logger.Infof(a, "Router %s not found routerInfo:%+v ", key, a.routerInfo)
	return nil, fmt.Errorf("router %s not found", key)
}

// Request 请求消息
type Request struct {
	TraceId     string                 `json:"trace_id"`     // 追踪ID
	RequestUser string                 `json:"request_user"` // 请求用户
	User        string                 `json:"user"`         // 应用所属用户
	App         string                 `json:"app"`          // 应用名
	Version     string                 `json:"version"`      // 版本号
	Method      string                 `json:"method"`       // 方法名（路径）
	Body        interface{}            `json:"body"`         // 请求体
	UrlQuery    map[string]interface{} `json:"url_query"`    // URL 查询参数
}

// Response 响应消息
type Response struct {
	Result interface{} `json:"result,omitempty"` // 结果
	Error  string      `json:"error,omitempty"`  // 错误信息
}

// Error 错误信息
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Subjects NATS 主题
type Subjects struct {
	AppRequest  string // 接收来自 app_runtime 的请求
	AppResponse string // 发送给 function_server 的响应
	Discovery   string // 接收发现消息
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
		natsURL = "nats://127.0.0.1:4223"
	}

	logger.Infof(context.Background(), "Connecting to NATS: %s", natsURL)
	conn, err := nats.Connect(natsURL)
	if err != nil {
		logger.Errorf(context.Background(), "Failed to connect to NATS: %v", err)
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}
	logger.Infof(context.Background(), "NATS connected successfully")

	newApp := &App{
		Context:    context.Background(),
		exit:       make(chan struct{}),
		conn:       conn,
		startTime:  time.Now(), // 记录启动时间
		routerInfo: make(map[string]*routerInfo),
		subjects: &Subjects{
			AppRequest:  subjects.BuildAppRuntime2AppSubject(env.User, env.App, env.Version),
			AppResponse: subjects.BuildApp2FunctionServerSubject(env.User, env.App, env.Version),
			Discovery:   subjects.GetRuntimeDiscoverySubject(),
		},
	}
	newApp.routerInfo[routerKey("/test/add", "POST")] = &routerInfo{
		Router:     "/test/add",
		Method:     "POST",
		HandleFunc: AddHandle,
		Template:   Temp,
	}
	newApp.routerInfo[routerKey("/test/get", "POST")] = &routerInfo{
		Router:     "/test/get",
		Method:     "POST",
		HandleFunc: GetHandle,
		Template:   Temp,
	}

	// 订阅请求消息
	logger.Infof(context.Background(), "Subscribing to app request: %s", newApp.subjects.AppRequest)
	subs, err := newApp.conn.Subscribe(newApp.subjects.AppRequest, newApp.handleMessage)
	if err != nil {
		logger.Errorf(context.Background(), "Failed to subscribe to app request: %v", err)
		return nil, fmt.Errorf("failed to subscribe to %s: %w", newApp.subjects.AppRequest, err)
	}
	newApp.subs = append(newApp.subs, subs)
	logger.Infof(context.Background(), "App request subscription successful")

	// 订阅发现消息
	logger.Infof(context.Background(), "Subscribing to discovery: %s", newApp.subjects.Discovery)
	discoverySub, err := newApp.conn.Subscribe(newApp.subjects.Discovery, newApp.handleDiscovery)
	if err != nil {
		logger.Errorf(context.Background(), "Failed to subscribe to discovery: %v", err)
		return nil, fmt.Errorf("failed to subscribe to discovery: %w", err)
	}
	newApp.subs = append(newApp.subs, discoverySub)
	logger.Infof(context.Background(), "Discovery subscription successful")

	// 订阅 runtime 关闭命令
	shutdownSubject := subjects.BuildRuntime2AppShutdownSubject(env.User, env.App, env.Version)
	logger.Infof(context.Background(), "Subscribing to runtime shutdown command: %s", shutdownSubject)
	shutdownSub, err := newApp.conn.Subscribe(shutdownSubject, newApp.handleShutdownCommand)
	if err != nil {
		logger.Errorf(context.Background(), "Failed to subscribe to shutdown command: %v", err)
		return nil, fmt.Errorf("failed to subscribe to shutdown command: %w", err)
	}
	newApp.subs = append(newApp.subs, shutdownSub)
	logger.Infof(context.Background(), "Runtime shutdown command subscription successful")

	logger.Infof(context.Background(), "NewApp() completed successfully")

	// 发送启动完成通知给 runtime
	// 通知 runtime 新版本已经成功启动并准备好接收请求
	if err := newApp.sendStartupNotification(); err != nil {
		logger.Warnf(context.Background(), "Failed to send startup notification: %v", err)
		// 不返回错误，启动通知失败不应阻止应用运行
	}

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
		// 发送关闭通知
		if err := a.sendCloseNotification(); err != nil {
			logger.Warnf(ctx, "Failed to send close notification: %v", err)
		}
		return ctx.Err()
	case <-a.exit:
		logger.Infof(ctx, "Exit signal received, shutting down...")
		// 发送关闭通知
		if err := a.sendCloseNotification(); err != nil {
			logger.Warnf(ctx, "Failed to send close notification: %v", err)
		}
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

	// 设置 trace_id header
	if resp.TraceId != "" {
		msg.Header.Set("trace_id", resp.TraceId)
	}
	msg.Header.Set("code", "0")

	if err := a.conn.PublishMsg(msg); err != nil {
		return
	}
}
func (a *App) sendErrResponse(resp *dto.RequestAppResp) {
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

	// 设置 trace_id header
	if resp.TraceId != "" {
		msg.Header.Set("trace_id", resp.TraceId)
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

	// 发送响应
	response := discovery.DiscoveryResponse{
		Type:      "response",
		User:      env.User,
		App:       env.App,
		Version:   env.Version,
		Status:    "running",
		RuntimeID: discoveryMsg.RuntimeID,
		StartTime: a.startTime, // 应用启动时间
		Timestamp: time.Now(),  // 响应时间
	}

	data, err := json.Marshal(response)
	if err != nil {
		return
	}

	// 发送到响应主题
	a.conn.Publish(subjects.GetAppDiscoveryResponseSubject(), data)
}

// Close 关闭应用
func (a *App) Close() error {
	logger.Infof(context.Background(), "App.Close() called")
	for _, sub := range a.subs {
		sub.Unsubscribe()
	}
	if a.conn != nil {
		a.conn.Close()
	}
	// 非阻塞方式发送退出信号
	select {
	case a.exit <- struct{}{}:
		logger.Infof(context.Background(), "Exit signal sent")
	default:
		logger.Infof(context.Background(), "Exit signal not sent (channel full)")
	}
	return nil
}

func Run() error {
	if app == nil {
		logger.Infof(context.Background(), "App is nil, initializing...")
		initApp()
	}
	logger.Infof(context.Background(), "Starting app...")
	err := app.Start(context.Background())
	if err != nil {
		logger.Errorf(context.Background(), "App.Start() failed: %v", err)
		return err
	}
	logger.Infof(context.Background(), "App.Start() returned successfully")
	return nil
}

// sendStartupNotification 发送启动完成通知
func (a *App) sendStartupNotification() error {
	// 构建启动通知消息
	notification := map[string]interface{}{
		"user":       env.User,
		"app":        env.App,
		"version":    env.Version,
		"status":     "started",
		"start_time": a.startTime.Format(time.RFC3339),
		"timestamp":  time.Now().Format(time.RFC3339),
	}

	data, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal startup notification: %w", err)
	}

	// 发送到启动通知主题
	subject := subjects.BuildAppStartupNotificationSubject(env.User, env.App, env.Version)
	if err := a.conn.Publish(subject, data); err != nil {
		return fmt.Errorf("failed to publish startup notification: %w", err)
	}

	logger.Infof(context.Background(), "Startup notification sent to subject: %s", subject)
	return nil
}

// sendCloseNotification 发送应用关闭通知
func (a *App) sendCloseNotification() error {
	// 构建关闭通知消息
	notification := map[string]interface{}{
		"user":       env.User,
		"app":        env.App,
		"version":    env.Version,
		"status":     "closed",
		"start_time": a.startTime.Format(time.RFC3339),
		"close_time": time.Now().Format(time.RFC3339),
		"timestamp":  time.Now().Format(time.RFC3339),
	}

	data, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal close notification: %w", err)
	}

	// 发送到关闭通知主题
	subject := subjects.BuildAppCloseNotificationSubject(env.User, env.App, env.Version)
	if err := a.conn.Publish(subject, data); err != nil {
		return fmt.Errorf("failed to publish close notification: %w", err)
	}

	logger.Infof(context.Background(), "Close notification sent to subject: %s", subject)
	return nil
}

// handleShutdownCommand 处理 runtime 发送的关闭命令
func (a *App) handleShutdownCommand(msg *nats.Msg) {
	ctx := context.Background()
	logger.Infof(ctx, "Received shutdown command from runtime: %s", string(msg.Data))

	// 设置关闭请求标志，拒绝新请求
	a.shutdownMu.Lock()
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

	// 发送关闭通知
	if err := a.sendCloseNotification(); err != nil {
		logger.Warnf(ctx, "Failed to send close notification: %v", err)
	}

	// 触发应用退出
	close(a.exit)
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
