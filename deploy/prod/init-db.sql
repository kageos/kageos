-- 生产 Compose 专用：官方共享源位于 deploy/base/infra/mysql/init-db.sql。
-- MySQL 首次启动挂载到 /docker-entrypoint-initdb.d/ 执行。
--
-- 密码由 compose 环境变量 MYSQL_ROOT_PASSWORD 注入；应用配置模板中的占位符由 entrypoint 替换。
--
-- 库与服务对应:
--   app_db       -> app-server
--   app-storage  -> app-storage
--   agent-server -> agent-server
--   hr-server    -> hr-server
--   hub          -> hub

CREATE DATABASE IF NOT EXISTS `app_db` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS `app-storage` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS `agent-server` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS `hr-server` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE DATABASE IF NOT EXISTS `hub` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
