# 生产恢复手册

这份手册只解决一个问题：**数据误删后，怎么尽快恢复**。

当前版本不追求复杂灾备编排，按下面的流程操作即可。

## 适用范围

当前 `backup-service` 支持恢复这三类数据：

- `namespace`：用户应用空间，适合误删某个用户 / 应用 / 目录后恢复
- `MySQL`：当前版本是**整库快照、整库恢复**
- `MinIO`：当前版本是**整仓对象快照、整仓恢复**

## 入口

- 控制台：`http://127.0.0.1:19088/backup`
- 值班速查：见 `RECOVERY_CHECKLIST.md`
- 健康检查：`GET /health`
- 任务列表：`GET /backup/api/v1/tasks`
- 快照列表：`GET /backup/api/v1/snapshots?resource_type=namespace|mysql|minio`

如果控制台打不开，先看：

```bash
go run ./cmd/aosctl logs --config deploy/prod/aos.yaml backup
```

## 恢复前确认

先确认这几件事：

1. `backup-service` 正常运行
2. 目标资源已经有快照
3. 这次恢复是否允许短时间对外进入维护页

推荐先做一次预检：

```bash
curl -X POST http://127.0.0.1:19088/backup/api/v1/precheck \
  -H 'Content-Type: application/json' \
  -d '{"requested_by":"operator","note":"manual recovery precheck"}'
```

## 通用恢复流程

无论恢复 `namespace`、`MySQL` 还是 `MinIO`，都按这个顺序来：

1. 打开控制台，确认要恢复的快照存在
2. 开启维护模式
3. 执行恢复
4. 等任务状态变成 `succeeded`
5. 手工验证业务
6. 关闭维护模式

### 开启维护模式

控制台里点“切换维护模式”即可，也可以直接调接口：

```bash
curl -X POST http://127.0.0.1:19088/backup/api/v1/maintenance \
  -H 'Content-Type: application/json' \
  -d '{
    "enabled": true,
    "requested_by": "operator",
    "reason": "manual recovery"
  }'
```

开启后，主站 Nginx 会自动返回 `503` 维护页。相关文件会写到：

- `/app/data/backup/state/maintenance.flag`
- `/app/data/backup/state/maintenance.html`
- `/app/data/backup/state/maintenance.json`

## 场景一：恢复 namespace

适用场景：

- 删了某个应用目录
- 删了某个用户空间
- 应用代码目录被误覆盖

### 操作方法

1. 在控制台的 `Namespace 快照` 区域选中目标快照
2. 开启维护模式
3. 点击“恢复选中 Namespace 快照”
4. 等待任务完成

接口方式：

```bash
curl http://127.0.0.1:19088/backup/api/v1/snapshots?resource_type=namespace
```

找到 `snapshot_id` 后执行：

```bash
curl -X POST http://127.0.0.1:19088/backup/api/v1/namespace/restore \
  -H 'Content-Type: application/json' \
  -d '{
    "requested_by": "operator",
    "note": "restore namespace snapshot",
    "snapshot_id": 1
  }'
```

### 恢复后验证

- 目标目录是否重新出现
- 目录内关键文件是否存在
- 页面能否正常打开
- 相关接口能否正常调用

## 场景二：恢复 MySQL

适用场景：

- 误删业务库
- 误清空表数据
- 明显的数据写坏，需要回到某次快照

### 当前边界

当前版本是**整库恢复**，不是单表恢复，也不是精准时间点恢复。  
如果误删的是一张表，当前也需要按整库快照回滚。

### 操作方法

1. 在控制台的 `MySQL 快照` 区域选中目标快照
2. 开启维护模式
3. 点击“恢复选中 MySQL 快照”
4. 等待任务完成

接口方式：

```bash
curl http://127.0.0.1:19088/backup/api/v1/snapshots?resource_type=mysql
```

找到 `snapshot_id` 后执行：

```bash
curl -X POST http://127.0.0.1:19088/backup/api/v1/mysql/restore \
  -H 'Content-Type: application/json' \
  -d '{
    "requested_by": "operator",
    "note": "restore mysql snapshot",
    "snapshot_id": 1
  }'
```

### 恢复后验证

- 登录是否正常
- 关键业务页面是否能打开
- 关键业务表数据是否恢复

## 场景三：恢复 MinIO

适用场景：

- 文件对象被误删
- 整个桶内容被误覆盖或异常清理

### 当前边界

当前版本是**整仓对象快照恢复**。  
如果只是误删了单个对象，当前仍按整次 MinIO 快照恢复处理。

### 操作方法

1. 在控制台的 `MinIO 快照` 区域选中目标快照
2. 开启维护模式
3. 点击“恢复选中 MinIO 快照”
4. 等待任务完成

接口方式：

```bash
curl http://127.0.0.1:19088/backup/api/v1/snapshots?resource_type=minio
```

找到 `snapshot_id` 后执行：

```bash
curl -X POST http://127.0.0.1:19088/backup/api/v1/minio/restore \
  -H 'Content-Type: application/json' \
  -d '{
    "requested_by": "operator",
    "note": "restore minio snapshot",
    "snapshot_id": 1
  }'
```

### 恢复后验证

- 文件是否重新可下载
- 页面引用的对象链接是否恢复
- 上传新文件是否正常

## 查看任务结果

恢复任务发起后，可以在控制台看，也可以查接口：

```bash
curl http://127.0.0.1:19088/backup/api/v1/tasks
curl http://127.0.0.1:19088/backup/api/v1/tasks/1
```

重点看：

- `status`
- `summary`
- `error_message`
- `detail`

## 恢复完成后

确认业务正常后，关闭维护模式：

```bash
curl -X POST http://127.0.0.1:19088/backup/api/v1/maintenance \
  -H 'Content-Type: application/json' \
  -d '{
    "enabled": false,
    "requested_by": "operator",
    "reason": "recovery finished"
  }'
```

## 建议的演练方式

上线前至少演练 3 次：

1. 删除一个测试 `namespace` 子目录，再恢复
2. 删除一张测试表里的数据，再恢复
3. 删除一个测试对象，再恢复 MinIO 快照

目标不是做复杂演练，而是确认两件事：

- 你们知道该点哪里
- 恢复完之后知道该检查什么

## 当前限制

- 现在还不是自动停服编排，只是维护模式挡入口流量
- `MySQL` 暂不支持单表恢复 / PITR
- `MinIO` 暂不支持单对象精细恢复
- 如果目标资源没有快照，就无法恢复
