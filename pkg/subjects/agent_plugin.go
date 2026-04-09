package subjects

import "fmt"

const (
	// agent / plugin
	AgentFunctionGenCommandSubject       = "agent.v1.cmd.function-gen"
	AgentFunctionGenCallbackReplySubject = "agent.v1.reply.function-gen.callback"
)

// BuildAgentMsgSubject 构建 agent 消息主题（用于 MsgSubject 字段）。
func BuildAgentMsgSubject(chatType, user string, agentID int64) string {
	return fmt.Sprintf("agent.v1.msg.%s.%s.%d", chatType, user, agentID)
}

// BuildPluginSubject 构建插件消息主题。
func BuildPluginSubject(user string, pluginID int64) string {
	return fmt.Sprintf("plugin.v1.msg.%s.%d", user, pluginID)
}
