# 添加用户类型和应用类型字段迁移说明

## 问题描述

在启动服务时，如果遇到以下错误：
```
Error 1054 (42S22): Unknown column 'type' in 'field list'
```

说明数据库表 `user` 和 `app` 中缺少 `type` 字段。

## 解决方案

### ⭐ 优先方案：使用 GORM AutoMigrate（推荐）

**正常情况下，GORM 的 AutoMigrate 应该能够自动添加字段。**

Model 中已经定义了 `Type` 字段：
- `core/hr-server/model/user.go` - `Type UserType`
- `core/app-server/model/app.go` - `Type AppType`

服务启动时会自动调用 `AutoMigrate`：
- `hr-server` 启动时会调用 `model.InitModels(db)`
- `app-server` 启动时会调用 `model.InitTables(db)`

**如果 AutoMigrate 没有自动添加字段，可能的原因：**
1. 服务启动时迁移失败但没有报错
2. GORM 版本问题
3. 数据库连接问题

**解决方法：**
1. 重启服务，让 AutoMigrate 重新执行
2. 检查服务启动日志，确认迁移是否成功
3. 如果仍然失败，使用下面的 SQL 迁移脚本作为备选方案

### 备选方案：使用 SQL 迁移脚本

如果 GORM AutoMigrate 无法自动添加字段，可以手动执行 SQL 迁移脚本。

## 执行方法

### 方法一：使用 mysql 命令行工具

```bash
# 假设数据库名为 "hr-server"，用户名 root，密码 root
mysql -uroot -proot "hr-server" < scripts/migration/add_user_app_type.sql

# 如果数据库名不同，请替换为实际的数据库名
# 例如：mysql -uroot -proot "your-database-name" < scripts/migration/add_user_app_type.sql
```

### 方法二：使用 MySQL 客户端

1. 连接到 MySQL 数据库
2. 选择对应的数据库（hr-server 和 app-server 的数据库）
3. 执行 SQL 脚本内容

### 方法三：分别执行（如果 hr-server 和 app-server 使用不同的数据库）

```bash
# 1. 为 hr-server 数据库添加 user.type 字段
mysql -uroot -proot "hr-server" < scripts/migration/add_user_app_type.sql

# 2. 为 app-server 数据库添加 app.type 字段
mysql -uroot -proot "app-server" < scripts/migration/add_user_app_type.sql
```

## 迁移内容

1. **user 表**：添加 `type` 字段（TINYINT，默认值 0）
   - 0: 普通用户
   - 1: 系统用户
   - 2: 智能体用户

2. **app 表**：添加 `type` 字段（TINYINT，默认值 0）
   - 0: 用户空间
   - 1: 系统空间

## 注意事项

- 执行前请备份数据库
- 如果表已存在 `type` 字段，脚本会报错，可以忽略或手动检查
- 迁移脚本是幂等的，但建议只执行一次

## 验证

执行后，可以通过以下 SQL 验证：

```sql
-- 检查 user 表的 type 字段
DESCRIBE `user`;

-- 检查 app 表的 type 字段
DESCRIBE `app`;
```

应该能看到 `type` 字段已添加。
