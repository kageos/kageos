# scripts/podman

本目录为**早期**用 `podman run` 单独起 MySQL/NATS/MinIO 的脚本，**后续不必维护**。

推荐统一用项目根目录的 **docker-compose.infra.yml** 拉起基础设施：

```bash
podman compose -f docker-compose.infra.yml up -d
```

容器名、账号、初始化（init-db.sql）与 app-runtime 看门狗一致，一键起齐 mysql8、nats-server、minio。

本目录脚本仅作备用参考，可保留不删。
