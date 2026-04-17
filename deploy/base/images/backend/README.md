# 旧后端大镜像（Legacy）

本目录保存的是一套历史上的“后端大镜像”方案：

- 单镜像内启动 `core-server`
- 可选再带 `hub-server`
- 容器内自带 Podman

它当前**不是** `deploy/prod` 的生产主线，也**不是** `deploy/dev` 的本地开发入口。

当前主线请看：

- 生产单机部署：`deploy/prod/`
- 本地开发：`deploy/dev/`
- 用户应用基础镜像：`deploy/base/images/app-base/`

只有在你明确要维护这条历史大镜像链路时，才需要改这里。
