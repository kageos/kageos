-- AI Agent OS 数据库初始化脚本
-- 由 MySQL 容器首次启动时自动执行（挂载到 /docker-entrypoint-initdb.d/）。
--
-- 账号密码（与 compose 和 configs 一致）:
--   user: root
--   password: root
-- compose 中: MYSQL_ROOT_PASSWORD: root
-- 各服务 configs 里 db.user / db.password 均为 root/root。
--
-- 各库与服务对应:
--   app_db       -> app-server
--   app-scheduler -> app-server scheduler
--   app-storage  -> app-storage
--   agent-server -> agent-server
--   hr-server    -> hr-server
--   hub          -> hub

CREATE DATABASE IF NOT EXISTS `app_db` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS `app-scheduler` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS `app-storage` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS `agent-server` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS `hr-server` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS `hub` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
