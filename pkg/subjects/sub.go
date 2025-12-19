package subjects

import (
	"fmt"
	"time"
)

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

// GetAppUpdateCallbackRequestSubject 获取 App Update Callback 请求主题
func GetAppUpdateCallbackRequestSubject(user, app, version string) string {
	return fmt.Sprintf("app.update.callback.%s.%s.%s", user, app, version)
}

// GetAppUpdateCallbackRequestSubjectPattern 获取 App Update Callback 请求主题模式（通配符）
func GetAppUpdateCallbackRequestSubjectPattern() string {
	return "app.update.callback.*.*.*"
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
	MessageTypeStatusShutdown    = "shutdown"    // 关闭命令
	MessageTypeStatusDiscovery   = "discovery"   // 服务发现
	MessageTypeStatusStartup     = "startup"     // 启动通知
	MessageTypeStatusClose       = "close"       // 关闭通知
	MessageTypeStatusOnAppUpdate = "onAppUpdate" // 当程序更新时候

	// Request/Reply 消息类型
	MessageTypeUpdateCallbackRequest = "update_callback_request" // 更新回调请求
)

// 消息结构体
type Message struct {
	ErrorMsg  string      `json:"error_msg"`
	Type      string      `json:"type"`
	User      string      `json:"user"`
	App       string      `json:"app"`
	Version   string      `json:"version"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

// GetAppRuntime2AppCreateRequestSubject 获取 app_runtime 到 app 创建请求的订阅主题
func GetAppRuntime2AppCreateRequestSubject() string {
	return "app_runtime.app.create"
}

// GetAppRuntime2AppUpdateRequestSubject 获取 app_runtime 到 app 更新请求的订阅主题
func GetAppRuntime2AppUpdateRequestSubject() string {
	return "app_runtime.app.update"
}

// GetAppRuntime2ServiceTreeCreateRequestSubject 获取 app_runtime 到 service_tree 创建请求的订阅主题
func GetAppRuntime2ServiceTreeCreateRequestSubject() string {
	return "app_runtime.service_tree.create"
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

// GetAppServer2AppRuntimeReadDirectoryFilesRequestSubject 获取 app_server 到 app_runtime 读取目录文件请求的订阅主题
func GetAppServer2AppRuntimeReadDirectoryFilesRequestSubject() string {
	return "app_server.app_runtime.read_directory_files"
}

// GetAppServer2AppRuntimeBatchCreateDirectoryTreeRequestSubject 获取 app_server 到 app_runtime 批量创建目录树请求的订阅主题
func GetAppServer2AppRuntimeBatchCreateDirectoryTreeRequestSubject() string {
	return "app_server.app_runtime.batch_create_directory_tree"
}

// GetAppServer2AppRuntimeUpdateServiceTreeRequestSubject 获取 app_server 到 app_runtime 更新服务树请求的订阅主题
func GetAppServer2AppRuntimeUpdateServiceTreeRequestSubject() string {
	return "app_server.app_runtime.update_service_tree"
}

// GetAppServer2AppRuntimeBatchWriteFilesRequestSubject 获取 app_server 到 app_runtime 批量写文件请求的订阅主题
func GetAppServer2AppRuntimeBatchWriteFilesRequestSubject() string {
	return "app_server.app_runtime.batch_write_files"
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

// ==================== Agent Server 相关主题 ====================

// BuildAgentMsgSubject 构建 agent 消息主题（用于 MsgSubject 字段）
// 格式：agent.{chat_type}.{user}.{id}
func BuildAgentMsgSubject(chatType, user string, agentID int64) string {
	return fmt.Sprintf("agent.%s.%s.%d", chatType, user, agentID)
}

// BuildAgentPluginSubject 构建 agent plugin 调用主题（别名，保持兼容）
// 格式：agent.{chat_type}.{user}.{id}
func BuildAgentPluginSubject(chatType, user string, agentID int64) string {
	return BuildAgentMsgSubject(chatType, user, agentID)
}

// BuildAgentPluginRunSubject 构建 agent plugin 执行主题（已废弃，使用 Plugin.Subject）
// 格式：agent.{chat_type}.{user}.{id}.run
// 注意：新架构中应该使用 Plugin.Subject，此函数保留用于向后兼容
func BuildAgentPluginRunSubject(chatType, user string, agentID int64) string {
	return fmt.Sprintf("agent.%s.%s.%d.run", chatType, user, agentID)
}

// BuildPluginSubject 构建插件主题
// 格式：plugins.{user}.{plugin_id}
func BuildPluginSubject(user string, pluginID int64) string {
	return fmt.Sprintf("plugins.%s.%d", user, pluginID)
}

// GetAgentServerFunctionGenSubject 获取 agent-server 函数生成结果队列主题
// 格式：agent_server.function_gen
func GetAgentServerFunctionGenSubject() string {
	return "agent_server.function_gen"
}

// GetAgentServerFunctionGenCallbackSubject 获取 agent-server 函数生成回调主题（app-server -> agent-server）
// 格式：agent_server.function_gen.callback
func GetAgentServerFunctionGenCallbackSubject() string {
	return "agent_server.function_gen.callback"
}

// ==================== Control Service 相关主题 ====================

// GetControlLicenseKeySubject 获取 Control Service License 密钥发布主题（推送模式）
// 格式：control.license.key
func GetControlLicenseKeySubject() string {
	return "control.license.key"
}

// GetControlLicenseKeyRequestSubject 获取 Control Service License 密钥请求主题（请求-响应模式）
// 格式：control.license.key.request
func GetControlLicenseKeyRequestSubject() string {
	return "control.license.key.request"
}

// GetControlLicenseKeyRefreshSubject 获取 Control Service License 密钥刷新主题（推送刷新指令）
// 格式：control.license.key.refresh
func GetControlLicenseKeyRefreshSubject() string {
	return "control.license.key.refresh"
}
