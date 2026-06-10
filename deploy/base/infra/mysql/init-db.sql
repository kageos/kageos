-- Kageos 数据库初始化脚本
-- 由 MySQL 容器首次启动时自动执行（挂载到 /docker-entrypoint-initdb.d/）。
--
-- 账号密码由 .kageos/dev/env/kageos.env 或生产部署 env 注入。
-- 本脚本只负责幂等创建服务数据库，不写入固定密码。
--
-- 各库与服务对应:
--   app-server   -> app-server
--   app-storage  -> app-storage
--   agent-server -> agent-server
--   connector-server -> connector-server
--   hr-server    -> hr-server
--   timer-scheduler -> timer-scheduler

CREATE DATABASE IF NOT EXISTS `app-server` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS `app-storage` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS `agent-server` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS `connector-server` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS `hr-server` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS `timer-scheduler` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
