# `deploy/customer/config/prod/`

客户 **主站** 一键部署专用 YAML（**非** Hub），与仓库 **`deploy/config/prod`**（开发/裸机）相互独立。

- 构建：`deploy/customer/Dockerfile` 将本目录复制到镜像内 **`/app/config.prod.template/`**。
- 运行：`entrypoint-main.sh` 每次启动复制到容器内 **`deploy/config/prod/`**，并对占位符做 `sed` 替换。

占位符：`__MYSQL_ROOT_PASSWORD__`、`__JWT_SECRET__`、`__CONTROL_ENC_KEY__`、`__MINIO_ROOT_USER__`、`__MINIO_ROOT_PASSWORD__`、`__SMTP_PASSWORD__`。
