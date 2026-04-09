# Backup Service

`backup-service` 是项目里的独立备份与恢复控制面。

它不挂在主站后台里，而是单独提供：

- 控制台页面：`/backup`
- 任务与状态 API：`/backup/api/v1/*`
- 健康检查：`/health`

当前目标不是做复杂灾备平台，而是解决一个更实际的问题：

- `namespace` 误删后能恢复
- `MySQL` 误删后能恢复
- `MinIO` 误删后能恢复

## 当前能力

当前已经支持：

- 环境预检
- 维护模式开关
- `namespace` 快照、恢复、删除、清理旧 `pre_restore`
- `MySQL` 快照、恢复、删除、清理旧 `pre_restore`
- `MinIO` 快照、恢复、删除、清理旧 `pre_restore`
- 最近任务、快照列表、路径状态查看
- 控制台最小 `Basic Auth`

当前恢复前都会自动创建一份 `pre_restore` 快照。

## 代码位置

- 入口：[main.go](/Users/beiluo/Documents/work/code/gitee.com/ai-agent-os/core/backup-service/cmd/app/main.go)
- 运行器：[runner.go](/Users/beiluo/Documents/work/code/gitee.com/ai-agent-os/core/backup-service/runner/runner.go)
- HTTP 服务：[server.go](/Users/beiluo/Documents/work/code/gitee.com/ai-agent-os/core/backup-service/server/server.go)
- 路由：[router.go](/Users/beiluo/Documents/work/code/gitee.com/ai-agent-os/core/backup-service/server/router.go)
- 控制面核心逻辑：[control_plane.go](/Users/beiluo/Documents/work/code/gitee.com/ai-agent-os/core/backup-service/service/control_plane.go)
- 控制台前端：[console.html](/Users/beiluo/Documents/work/code/gitee.com/ai-agent-os/core/backup-service/server/assets/console.html)

## 开发环境

开发环境配置在：

- [backup-service.yaml](/Users/beiluo/Documents/work/code/gitee.com/ai-agent-os/deploy/dev/config/backup-service.yaml)

先起基础设施：

```bash
bash deploy/dev/scripts/infra.sh up
```

再启动服务：

```bash
APP_ENV=dev go run ./core/backup-service/cmd/app
```

然后打开：

```text
http://127.0.0.1:19088/backup
```

当前 dev 默认账号：

- 用户名：`admin`
- 密码：`admin123`

说明：

- 根目录会通过仓库里的 `.ai-agent-os-root` 自动识别，不需要再传 `AI_AGENT_OS_ROOT`
- dev 下如果本机没有 `mysql` / `mysqldump`，会优先尝试进入本地 dev MySQL 容器执行

## 生产环境

生产配置模板在：

- [backup-service.yaml](/Users/beiluo/Documents/work/code/gitee.com/ai-agent-os/deploy/prod/config/template/backup-service.yaml)

生产部署文档见：

- [README.md](/Users/beiluo/Documents/work/code/gitee.com/ai-agent-os/deploy/prod/README.md)
- [RECOVERY.md](/Users/beiluo/Documents/work/code/gitee.com/ai-agent-os/deploy/prod/RECOVERY.md)
- [RECOVERY_CHECKLIST.md](/Users/beiluo/Documents/work/code/gitee.com/ai-agent-os/deploy/prod/RECOVERY_CHECKLIST.md)

当前 prod 约定：

- `backup-service` 是独立控制面
- 监听 `127.0.0.1:19088`
- 存储根目录统一走 `STORAGE_ROOT`

重要目录：

- `${STORAGE_ROOT}/data/backup/repo`
- `${STORAGE_ROOT}/data/backup/state`
- `${STORAGE_ROOT}/data/backup/staging`

状态库：

- `${STORAGE_ROOT}/data/backup/state/backup-service.db`

## 常用接口

- `GET /health`
- `GET /backup/api/v1/status`
- `GET /backup/api/v1/tasks`
- `GET /backup/api/v1/snapshots?resource_type=namespace|mysql|minio`
- `POST /backup/api/v1/precheck`
- `POST /backup/api/v1/maintenance`
- `POST /backup/api/v1/namespace/snapshots`
- `POST /backup/api/v1/namespace/restore`
- `POST /backup/api/v1/mysql/snapshots`
- `POST /backup/api/v1/mysql/restore`
- `POST /backup/api/v1/minio/snapshots`
- `POST /backup/api/v1/minio/restore`
- `DELETE /backup/api/v1/snapshots/:id`
- `POST /backup/api/v1/snapshots/prune`

## 建议操作顺序

日常手工恢复，按这个顺序即可：

1. 打开控制台，先看目标资源是否已有快照
2. 执行一次预检
3. 开启维护模式
4. 选择快照并恢复
5. 验证业务
6. 关闭维护模式

## 当前边界

这版是可用的恢复工具，不是完整灾备系统。

当前边界：

- `MySQL` 还是整库快照、整库恢复
- `MinIO` 还是整仓对象快照、整仓恢复
- 维护模式当前是“挡入口流量”，不是完整停机编排
- 没有做多角色权限和审批流
- 重点是“误删后能救回来”，不是做复杂灾备平台
