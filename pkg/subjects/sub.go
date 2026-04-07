package subjects

import (
	"fmt"
	"time"
)

// NATS 主题命名规范：
// 1. 统一使用点分隔：<target>.<version>.<kind>.<domain>.<action>[.<scope...>]。
// 2. target 表示该主题主要由谁消费/归谁负责，例如 runtime、app、gateway、control。
// 3. version 当前固定为 v1；后续协议升级时新增独立版本层，例如 v2。
// 4. kind 取值固定为 cmd / query / event / reply：
//    - cmd: 目标方执行命令或变更
//    - query: 目标方响应查询或 request-reply
//    - event: 目标方观察生命周期/广播事件
//    - reply: 目标方接收异步回复
// 5. 静态 token 一律使用小写 kebab-case；动态 scope 放在尾部，当前主要是 {user}.{app}.{version}。

const (
	// runtime / app 调用链
	RuntimeAppCreateCommandSubject                = "runtime.v1.cmd.app.create"
	RuntimeAppUpdateCommandSubject                = "runtime.v1.cmd.app.update"
	RuntimeAppDeleteCommandSubject                = "runtime.v1.cmd.app.delete"
	RuntimeServiceTreeCreateCommandSubject        = "runtime.v1.cmd.service-tree.create"
	RuntimeServiceTreeDeleteCommandSubject        = "runtime.v1.cmd.service-tree.delete"
	RuntimeServiceTreeUpdateCommandSubject        = "runtime.v1.cmd.service-tree.update"
	RuntimeDirectoryFilesReadQuerySubject         = "runtime.v1.query.directory-files.read"
	RuntimeFileReplaceBatchCommandSubject         = "runtime.v1.cmd.file.replace-batch"
	RuntimeFileDeleteCommandSubject               = "runtime.v1.cmd.file.delete"
	RuntimeAppLogReadQuerySubject                 = "runtime.v1.query.app-log.read"
	RuntimeDirectoryTreeBatchCreateCommandSubject = "runtime.v1.cmd.directory-tree.batch-create"
	RuntimeFileBatchWriteCommandSubject           = "runtime.v1.cmd.file.batch-write"
	RuntimeNamespaceCreateCommandSubject          = "runtime.v1.cmd.namespace.create"
	RuntimeAppInvokeCommandSubjectPattern         = "runtime.v1.cmd.app.invoke.*.*.*"
	RuntimeLifecycleEventSubjectPattern           = "runtime.v1.event.lifecycle.*.*.*"

	AppControlSubjectPattern              = "app.v1.cmd.control.*.*.*"
	AppDiscoveryRequestSubject            = "app.v1.cmd.discovery.request"
	AppServerAppInvokeReplySubjectPattern = "app-server.v1.reply.app.invoke.*.*.*"

	// agent / plugin
	AgentFunctionGenCommandSubject       = "agent.v1.cmd.function-gen"
	AgentFunctionGenCallbackReplySubject = "agent.v1.reply.function-gen.callback"

	// license / control
	ControlLicenseKeyGetQuerySubject = "control.v1.query.license-key.get"
	LicenseKeyUpdatedEventSubject    = "license.v1.event.key.updated"
	LicenseKeyRefreshEventSubject    = "license.v1.event.key.refresh"

	// message / gateway
	MessageSendCommandSubject                 = "message.v1.cmd.send"
	MessageSendQueueGroup                     = MessageSendCommandSubject
	GatewayTokenInvalidateCommandSubject      = "gateway.v1.cmd.token.invalidate"
	GatewayTokenRemoveBlacklistCommandSubject = "gateway.v1.cmd.token.remove-blacklist"
)

// BuildAppInvokeSubject 构建 runtime -> app 的调用主题。
func BuildAppInvokeSubject(user, app, version string) string {
	return fmt.Sprintf("app.v1.cmd.invoke.%s.%s.%s", user, app, version)
}

// BuildAppServerAppInvokeReplySubject 构建 app -> app-server 的异步回复主题。
func BuildAppServerAppInvokeReplySubject(user, app, version string) string {
	return fmt.Sprintf("app-server.v1.reply.app.invoke.%s.%s.%s", user, app, version)
}

// BuildAppControlSubject 构建 runtime -> app 的控制主题。
// 当前用于 shutdown 与 onAppUpdate request-reply。
func BuildAppControlSubject(user, app, version string) string {
	return fmt.Sprintf("app.v1.cmd.control.%s.%s.%s", user, app, version)
}

// BuildRuntimeLifecycleEventSubject 构建 app -> runtime 的生命周期事件主题。
func BuildRuntimeLifecycleEventSubject(user, app, version string) string {
	return fmt.Sprintf("runtime.v1.event.lifecycle.%s.%s.%s", user, app, version)
}

// BuildRuntimeAppInvokeCommandSubject 构建 app-server -> runtime 的应用调用主题。
func BuildRuntimeAppInvokeCommandSubject(user, app, version string) string {
	return fmt.Sprintf("runtime.v1.cmd.app.invoke.%s.%s.%s", user, app, version)
}

// BuildAgentMsgSubject 构建 agent 消息主题（用于 MsgSubject 字段）。
func BuildAgentMsgSubject(chatType, user string, agentID int64) string {
	return fmt.Sprintf("agent.v1.msg.%s.%s.%d", chatType, user, agentID)
}

// BuildPluginSubject 构建插件消息主题。
func BuildPluginSubject(user string, pluginID int64) string {
	return fmt.Sprintf("plugin.v1.msg.%s.%d", user, pluginID)
}

// 消息类型常量
const (
	MessageTypeStatusShutdown    = "shutdown"
	MessageTypeStatusDiscovery   = "discovery"
	MessageTypeStatusStartup     = "startup"
	MessageTypeStatusClose       = "close"
	MessageTypeStatusOnAppUpdate = "onAppUpdate"

	MessageTypeUpdateCallbackRequest = "update_callback_request"
)

// Message 为 runtime / app 生命周期与控制链路共用的统一消息体。
type Message struct {
	ErrorMsg  string      `json:"error_msg"`
	Type      string      `json:"type"`
	User      string      `json:"user"`
	App       string      `json:"app"`
	Version   string      `json:"version"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}
