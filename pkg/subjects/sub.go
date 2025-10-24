package subjects

import "fmt"

//整个流程是function_server给app_runtime发消息，如果对应的容器没启动，则启动容器，

// BuildAppRuntime2AppSubject 构建 app_runtime 到 app 的具体主题
func BuildAppRuntime2AppSubject(user, app, version string) string {
	return fmt.Sprintf("app_runtime.app.%s.%s.%s", user, app, version)
}

// BuildApp2FunctionServerSubject 构建 app 到 function_server 的具体主题
func BuildApp2FunctionServerSubject(user, app, version string) string {
	return fmt.Sprintf("app.function_server.%s.%s.%s", user, app, version)
}

// GetApp2FunctionServerResponseSubject 获取 app 到 function_server 响应的订阅主题（通配符）
func GetApp2FunctionServerResponseSubject() string {
	return "app.function_server.*.*.*"
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

// GetAppDiscoveryResponseSubject 获取 app 响应发现的主题
func GetAppDiscoveryResponseSubject() string {
	return "ai-agent-os.app.discovery.response"
}

// GetAppServer2AppRuntimeDeleteRequestSubject 获取 app_server 到 app_runtime 删除请求的订阅主题
func GetAppServer2AppRuntimeDeleteRequestSubject() string {
	return "app_server.app_runtime.delete"
}

// BuildAppStartupNotificationSubject 构建应用启动完成通知的主题
func BuildAppStartupNotificationSubject(user, app, version string) string {
	return fmt.Sprintf("app.startup.notification.%s.%s.%s", user, app, version)
}

// GetAppStartupNotificationSubject 获取应用启动完成通知的订阅主题（通配符）
func GetAppStartupNotificationSubject() string {
	return "app.startup.notification.*.*.*"
}

// BuildAppCloseNotificationSubject 构建应用关闭通知的主题
func BuildAppCloseNotificationSubject(user, app, version string) string {
	return fmt.Sprintf("app.close.notification.%s.%s.%s", user, app, version)
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
