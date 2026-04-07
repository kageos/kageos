# 值班恢复清单

出事先看这个，不要先翻长文档。

## 入口

- 控制台：`http://127.0.0.1:19088/backup`
- 长手册：[RECOVERY.md](/Users/beiluo/Documents/work/code/gitee.com/ai-agent-os/deploy/prod/RECOVERY.md)
- 看 backup 日志：`bash build.sh logs backup`

## 10 步走完

1. 先确认 `backup-service` 控制台能打开。
2. 先看有没有目标快照。
3. 先执行一次预检。
4. 开启维护模式。
5. 选对资源类型：`namespace` / `MySQL` / `MinIO`。
6. 选对快照，再点恢复。
7. 等任务状态变成 `succeeded`。
8. 验证业务是否恢复。
9. 关闭维护模式。
10. 记录这次恢复用了哪个快照、恢复了什么。

## 预检

```bash
curl -X POST http://127.0.0.1:19088/backup/api/v1/precheck \
  -H 'Content-Type: application/json' \
  -d '{"requested_by":"operator","note":"manual recovery precheck"}'
```

## 开维护模式

```bash
curl -X POST http://127.0.0.1:19088/backup/api/v1/maintenance \
  -H 'Content-Type: application/json' \
  -d '{"enabled":true,"requested_by":"operator","reason":"manual recovery"}'
```

## 查快照

```bash
curl http://127.0.0.1:19088/backup/api/v1/snapshots?resource_type=namespace
curl http://127.0.0.1:19088/backup/api/v1/snapshots?resource_type=mysql
curl http://127.0.0.1:19088/backup/api/v1/snapshots?resource_type=minio
```

## 恢复 namespace

```bash
curl -X POST http://127.0.0.1:19088/backup/api/v1/namespace/restore \
  -H 'Content-Type: application/json' \
  -d '{"requested_by":"operator","note":"restore namespace snapshot","snapshot_id":1}'
```

恢复后看：

- 目录回来了没有
- 关键文件在不在
- 页面和接口正常不正常

## 恢复 MySQL

```bash
curl -X POST http://127.0.0.1:19088/backup/api/v1/mysql/restore \
  -H 'Content-Type: application/json' \
  -d '{"requested_by":"operator","note":"restore mysql snapshot","snapshot_id":1}'
```

恢复后看：

- 能不能登录
- 关键业务表数据回来了没有
- 核心页面能不能打开

## 恢复 MinIO

```bash
curl -X POST http://127.0.0.1:19088/backup/api/v1/minio/restore \
  -H 'Content-Type: application/json' \
  -d '{"requested_by":"operator","note":"restore minio snapshot","snapshot_id":1}'
```

恢复后看：

- 文件能不能下载
- 页面里的资源能不能打开
- 新上传正不正常

## 看任务结果

```bash
curl http://127.0.0.1:19088/backup/api/v1/tasks
curl http://127.0.0.1:19088/backup/api/v1/tasks/1
```

重点看：

- `status`
- `summary`
- `error_message`

## 关维护模式

```bash
curl -X POST http://127.0.0.1:19088/backup/api/v1/maintenance \
  -H 'Content-Type: application/json' \
  -d '{"enabled":false,"requested_by":"operator","reason":"recovery finished"}'
```

## 先记住这 3 句话

- 没快照就恢复不了。
- 先开维护模式，再恢复。
- 恢复成功不等于业务正常，必须自己点一遍。
