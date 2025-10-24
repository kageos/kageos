package subjects

import "fmt"

// 重构后的主题设计：保持应用请求/响应独立，简化状态通知主题

// 保持独立的复杂主题
// BuildAppRuntime2AppSubject 构建 app_runtime 到 app 的具体主题（应用请求）
func BuildAppRuntime2AppSubject(user, app, version string) string {
	return fmt.Sprintf("app_runtime.app.%s.%s.%s", user, app, version)
}

// BuildApp2FunctionServerSubject 构建 app 到 function_server 的具体主题（应用响应）
func BuildApp2FunctionServerSubject(user, app, version string) string {
	return fmt.Sprintf("app.function_server.%s.%s.%s", user, app, version)
}

// GetApp2FunctionServerResponseSubject 获取 app 到 function_server 响应的订阅主题（通配符）
func GetApp2FunctionServerResponseSubject() string {
	return "app.function_server.*.*.*"
}

// 简化的状态通知主题
// BuildAppStatusSubject 构建 SDK App 状态主题
// 处理：shutdown、discovery
func BuildAppStatusSubject(user, app, version string) string {
	return fmt.Sprintf("app.status.%s.%s.%s", user, app, version)
}

// GetAppStatusSubjectPattern 获取 SDK App 状态主题模式（通配符）
func GetAppStatusSubjectPattern() string {
	return "app.status.*.*.*"
}

// BuildRuntimeStatusSubject 构建 Runtime 状态主题
// 处理：startup、close、discovery
func BuildRuntimeStatusSubject(user, app, version string) string {
	return fmt.Sprintf("runtime.status.%s.%s.%s", user, app, version)
}

// GetRuntimeStatusSubjectPattern 获取 Runtime 状态主题模式（通配符）
func GetRuntimeStatusSubjectPattern() string {
	return "runtime.status.*.*.*"
}

// 消息类型常量
const (
	// 状态通知消息类型
	MessageTypeShutdown  = "shutdown"  // 关闭命令
	MessageTypeDiscovery = "discovery" // 服务发现
	MessageTypeStartup   = "startup"   // 启动通知
	MessageTypeClose     = "close"     // 关闭通知
)

// 消息结构体
type Message struct {
	Type      string      `json:"type"`
	User      string      `json:"user"`
	App       string      `json:"app"`
	Version   string      `json:"version"`
	Data      interface{} `json:"data"`
	Timestamp string      `json:"timestamp"`
}

// GetAppRuntime2AppCreateRequestSubject 获取 app_runtime 到 app 创建请求的订阅主题
func GetAppRuntime2AppCreateRequestSubject() string {
	return "app_runtime.app.create"
}

// GetAppRuntime2AppUpdateRequestSubject 获取 app_runtime 到 app 更新请求的订阅主题
func GetAppRuntime2AppUpdateRequestSubject() string {
	return "app_runtime.app.update"
}

// GetFunctionServer2AppRuntimeNamespaceCreateSubject 获取 function_server 到 app_runtime 命名空间创建的主题
func GetFunctionServer2AppRuntimeNamespaceCreateSubject() string {
	return "function_server.app_runtime.namespace.create"
}

// BuildFunctionServer2AppRuntimeSubject 构建 function_server 到 app_runtime 的具体主题
func BuildFunctionServer2AppRuntimeSubject(user, app, version string) string {
	return fmt.Sprintf("function_server.app_runtime.%s.%s.%s", user, app, version)
}

// GetFunctionServer2AppRuntimeRequestSubject 获取 function_server 到 app_runtime 请求的订阅主题（通配符）
func GetFunctionServer2AppRuntimeRequestSubject() string {
	return "function_server.app_runtime.*.*.*"
}

// GetRuntimeDiscoverySubject 获取 runtime 发现应用的广播主题
func GetRuntimeDiscoverySubject() string {
	return "ai-agent-os.runtime.discovery"
}

// GetAppServer2AppRuntimeDeleteRequestSubject 获取 app_server 到 app_runtime 删除请求的订阅主题
func GetAppServer2AppRuntimeDeleteRequestSubject() string {
	return "app_server.app_runtime.delete"
}

// GetAppStartupNotificationSubject 获取应用启动完成通知的订阅主题（通配符）
func GetAppStartupNotificationSubject() string {
	return "app.startup.notification.*.*.*"
}

// GetAppCloseNotificationSubject 获取应用关闭通知的订阅主题（通配符）
func GetAppCloseNotificationSubject() string {
	return "app.close.notification.*.*.*"
}

// BuildRuntime2AppShutdownSubject 构建 runtime 到 app 的关闭命令主题
func BuildRuntime2AppShutdownSubject(user, app, version string) string {
	return fmt.Sprintf("runtime.app.shutdown.%s.%s.%s", user, app, version)
}

// GetRuntime2AppShutdownSubject 获取 runtime 到 app 关闭命令的订阅主题（通配符）
func GetRuntime2AppShutdownSubject() string {
	return "runtime.app.shutdown.*.*.*"
}
