# Prod Quick Start

生产部署入口是 Go 部署器 `aosctl`。Compose 仍是底层容器执行器，但用户不需要直接维护 Compose 文件。

## 前提

- Linux
- 已安装 `podman compose` 或 `docker compose`
- `/data` 可写
- `80` 端口空闲；如果本机终止 HTTPS，`443` 也要空闲

## 首次部署

在仓库根目录执行：

```bash
go run ./cmd/aosctl init --base-url http://your-ip-or-domain
go run ./cmd/aosctl doctor --config deploy/prod/aos.yaml
go run ./cmd/aosctl up --config deploy/prod/aos.yaml
go run ./cmd/aosctl verify --config deploy/prod/aos.yaml
```

访问：

```text
http://your-ip-or-domain
```

## 常用命令

```bash
go run ./cmd/aosctl status --config deploy/prod/aos.yaml
go run ./cmd/aosctl logs --config deploy/prod/aos.yaml main
go run ./cmd/aosctl down --config deploy/prod/aos.yaml
```

## 生成物

生成物位于 `deploy/prod/.generated/`，不要手工编辑；需要变更时修改 `deploy/prod/aos.yaml` 后重新执行 `aosctl up`。
