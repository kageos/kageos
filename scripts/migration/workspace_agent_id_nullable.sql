-- ============================================
-- 工作台会话/消息：agent_id 允许 NULL
-- 工作台会话无关联智能体，插入 agent_id=NULL 以避免 FK 违反 (agents.id 无 0)
--
-- 使用：mysql -u<user> -p <agent-server> < scripts/migration/workspace_agent_id_nullable.sql
-- ============================================

ALTER TABLE `agent_chat_sessions`
  MODIFY COLUMN `agent_id` BIGINT NULL COMMENT '智能体ID，工作台会话可为空';

ALTER TABLE `agent_chat_messages`
  MODIFY COLUMN `agent_id` BIGINT NULL COMMENT '智能体ID，工作台消息可为空';
