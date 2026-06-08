# 部署总览（官方入口）

> 状态：执行口径
> 更新时间：2026-05-17
> 负责人窗口：事项 5 / codex/local-dev-onboarding

本目录只保留当前 **官方部署入口**、**共享交付资源** 和 **可选安全模块**。
如果你是第一次看这个仓库，先不要在根目录乱找脚本，直接按下面选入口。

## 先看哪一个

| 你的场景 | 先看哪里 | 说明 |
|------|------|------|
| 本地开发 | [dev/README.md](dev/README.md) | 起本地 MySQL / NATS / MinIO，跑后端源码和前端 |
| 单机生产部署 | [prod/README.md](prod/README.md) | 当前官方生产入口，基于 Compose |
| 只想一眼看懂怎么部署 | [prod/QUICK_START.md](prod/QUICK_START.md) | 最短路径，直接照抄 |
| 只想快速部署 | [prod/DEPLOY_TUTORIAL.md](prod/DEPLOY_TUTORIAL.md) | 一分钟部署版 |
| 看部署分层与依赖图 | [prod/README.md#部署分层](prod/README.md#部署分层) | 单机生产部署的分层、依赖和排障心智模型 |
| 看生命周期 SOP | [../docs/kagectl-lifecycle-sop.md](../docs/kagectl-lifecycle-sop.md) | dev/prod 两种模式的统一 `kagectl` 入口 |
| 找共享资源 | [base/README.md](base/README.md) | canonical Dockerfile、init SQL、共享脚本都在这里 |
| 做容器防删限制 | [security/README.md](security/README.md) | 可选的 AppArmor / SELinux 安装资源 |

## 部署方式选择

| 方式 | 适合场景 | 入口 | 说明 |
|------|------|------|------|
| `deploy/dev` | 本地开发、联调、排查问题 | `go run ./cmd/kagectl bootstrap --dev` | 基础设施容器化，后端源码运行，前端本地 `npm run dev` |
| `deploy/prod` 本地构建 | 单机测试环境、演示环境、还没有镜像发布链时的正式部署 | `go run ./cmd/kagectl init` → `up` | Go 部署器渲染配置并调用 Compose 构建/启动 |
| `deploy/prod` 发布镜像 | 企业生产、固定 tag、需要可回滚的环境 | `go run ./cmd/kagectl up --image` | 目标机不做主镜像源码构建，直接拉 `images.main` 并初始化运行时底座 |
| `deploy/prod` HTTP | 内网部署、临时验证 | `site.tls_mode=http` | 容器内只跑 HTTP |
| `deploy/prod` 外部 TLS | 已有 LB / CDN / WAF / Ingress 做 TLS 终止 | `site.tls_mode=external` | 容器内只跑 HTTP，HTTPS 由外层代理处理 |
| `deploy/prod` 内建 HTTPS | 单机公网直出、自己持有证书 | `site.tls_mode=redirect` | 证书内容通过环境变量或 `kage.yaml` 注入，容器内 Nginx 直接提供 HTTPS |

当前成熟主线只有两条：

- 开发：`deploy/dev`
- 单机生产：`deploy/prod`

其中 `deploy/prod` 只是同时支持“本地构建发布”和“预构建镜像发布”两种方式；多机分布式目前还不是官方成熟入口。

## 当前结构

```text
deploy/
  README.md
  dev/        # 本地开发入口
  prod/       # 单机生产入口
  base/       # dev / prod 共享资源
  security/   # 可选安全策略
```

## 各目录职责

### `dev/`

- 面向开发同学
- 重点是“改代码快、起环境快、能本地调试”
- 生命周期入口：
  - `cmd/kagectl`
- 后端本地开发入口：
  - `go run ./cmd/kagectl up`
  - 或 GoLand 启动 `core/cmd/main/main.go`，模式由 `.kageos/kageos.env` 决定
- 用户应用运行时基础镜像：
  - `deploy/base/scripts/build-app-base-image.sh`

### `prod/`

- 面向单机生产部署
- 重点是“部署器清楚、配置清楚、生成物清楚、Compose 只做底层执行”
- 入口文件：
  - `cmd/kagectl`
  - `deploy/prod/kage.example.yaml`
  - `deploy/prod/QUICK_START.md`

### `base/`

- 不直接当部署入口
- 只放共享 canonical 资源
- 当前主要包含：
  - `images/app-base/`
  - `infra/mysql/`
  - `infra/nats/`
  - `scripts/`

### `security/`

- 可选模块，不是默认必装
- 给需要内核级容器防删限制的环境用
- 包含：
  - `apparmor/`
  - `selinux/`

## 当前约定

- 本地开发优先走 `deploy/dev/`
- 单机生产优先走 `deploy/prod/`
- 共享资源只在 `deploy/base/` 维护，不要在 `dev/`、`prod/` 各复制一份
- 旧实验性部署思路已经不作为主线入口，后续不要再额外引入平行部署目录

## 补充

- 只跑前端、连接远端网关：看仓库根目录的 `前端开发-本地与连线上.md`
- 用户应用运行时基础镜像的 canonical 构建脚本：`deploy/base/scripts/build-app-base-image.sh`
- 单机生产的直接操作教程：`deploy/prod/DEPLOY_TUTORIAL.md`
