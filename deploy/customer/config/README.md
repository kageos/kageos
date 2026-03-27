# `deploy/customer/config/prod/`

这是旧版 `deploy/customer/` 生产部署的**兼容模板目录**。

官方生产模板目录已迁到 **`deploy/prod/config/template/`**；本目录当前仅服务于旧的 `deploy/customer/Dockerfile` / `entrypoint-main.sh` 路径。

- 构建：`deploy/customer/Dockerfile` 将本目录复制到镜像内 **`/app/config.prod.template/`**。
- 运行：`deploy/customer/entrypoint-main.sh` 每次启动渲染到容器内 **`deploy/config/prod/`**。

模板变量：`${MYSQL_ROOT_PASSWORD}`、`${JWT_SECRET}`、`${CONTROL_ENC_KEY}`、`${MINIO_ROOT_USER}`、`${MINIO_ROOT_PASSWORD}`、`${SMTP_PASSWORD}` 等；由 `entrypoint-main.sh` 用 `envsubst` 渲染。
