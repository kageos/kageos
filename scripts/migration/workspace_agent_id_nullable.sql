-- ============================================
-- 历史兼容迁移：旧库的工作台会话/消息表可能仍带 agent_id NOT NULL。
-- 新模型不再创建 agent_id；仅老库升级时需要执行此脚本，避免历史列阻塞写入。
--
-- 使用：mysql -u<user> -p <agent-server> < scripts/migration/workspace_agent_id_nullable.sql
-- ============================================

ALTER TABLE `agent_chat_sessions`
  MODIFY COLUMN `agent_id` BIGINT NULL COMMENT '历史兼容字段，工作台不再写入';

ALTER TABLE `agent_chat_messages`
  MODIFY COLUMN `agent_id` BIGINT NULL COMMENT '历史兼容字段，工作台不再写入';
