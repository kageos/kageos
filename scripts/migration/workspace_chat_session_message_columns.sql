-- ============================================
-- 智能工作台：agent_chat_sessions / agent_chat_messages 扩展字段
-- - agent_chat_sessions: full_code_path, source（用于 workspace 会话，full_code_path 有语意；source=workspace 区分）
-- - agent_chat_messages: tool_calls, tool_call_id（用于 Tool 循环：assistant 的 tool_calls，tool 消息的 tool_call_id）
--
-- 使用方法：
--   mysql -u<user> -p <agent-server> < scripts/migration/workspace_chat_session_message_columns.sql
-- ============================================

-- agent_chat_sessions: 增加 full_code_path、source
ALTER TABLE `agent_chat_sessions`
  ADD COLUMN `full_code_path` VARCHAR(512) NULL COMMENT '服务目录完整路径（workspace 用，有语意）' AFTER `tree_id`,
  ADD COLUMN `source` VARCHAR(32) NULL COMMENT '来源：workspace=工作台，空=function_gen' AFTER `full_code_path`;

-- agent_chat_messages: 增加 tool_calls、tool_call_id（支持 role=tool 与 assistant 的 tool_calls）
ALTER TABLE `agent_chat_messages`
  ADD COLUMN `tool_calls` JSON NULL COMMENT 'assistant 的 tool_calls（LLM 返回）' AFTER `files`,
  ADD COLUMN `tool_call_id` VARCHAR(64) NULL COMMENT 'role=tool 时的 tool_call_id' AFTER `tool_calls`;
