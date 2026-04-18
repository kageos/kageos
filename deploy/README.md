# 部署总览（官方入口）

本目录只保留当前 **官方部署入口**、**共享交付资源** 和 **可选安全模块**。  
如果你是第一次看这个仓库，先不要在根目录乱找脚本，直接按下面选入口。

## 先看哪一个

| 你的场景 | 先看哪里 | 说明 |
|------|------|------|
| 本地开发 | [dev/README.md](dev/README.md) | 起本地 MySQL / NATS / MinIO，跑后端源码和前端 |
| 单机生产部署 | [prod/README.md](prod/README.md) | 当前官方生产入口，基于 Compose |
| 只想一眼看懂怎么部署 | [prod/QUICK_START.md](prod/QUICK_START.md) | 最短路径，直接照抄 |
| 只想快速部署 | [prod/DEPLOY_TUTORIAL.md](prod/DEPLOY_TUTORIAL.md) | 一分钟部署版 |
| 看启动顺序与依赖图 | [prod/DEPLOYMENT_FLOW.md](prod/DEPLOYMENT_FLOW.md) | 单机生产部署的依赖、启动顺序、流程图 |
| 找共享资源 | [base/README.md](base/README.md) | canonical Dockerfile、init SQL、共享脚本都在这里 |
| 做容器防删限制 | [security/README.md](security/README.md) | 可选的 AppArmor / SELinux 安装资源 |

## 部署方式选择

| 方式 | 适合场景 | 入口 | 说明 |
|------|------|------|------|
| `deploy/dev` | 本地开发、联调、排查问题 | `bash deploy/dev/scripts/infra.sh up` + `bash deploy/dev/scripts/run-backend.sh` | 基础设施容器化，后端源码运行，前端本地 `npm run dev` |
| `deploy/prod` 本地构建 | 单机测试环境、演示环境、还没有镜像发布链时的正式部署 | `bash deploy/prod/build.sh init` → `up` | 目标机本地构建 `agentos-main`，再显式准备 `APP_BASE_IMAGE` 后启动；`doctor` 只在想看显式预检时再手动跑 |
| `deploy/prod` 发布镜像 | 企业生产、固定 tag、需要可回滚的环境 | `bash deploy/prod/build.sh init --image` → `up` | 目标机不做主镜像源码构建，直接拉 `MAIN_IMAGE` 并初始化运行时底座；`doctor` 可选 |
| `deploy/prod` HTTP | 内网部署、临时验证 | `TLS_MODE=http` | 容器内只跑 HTTP |
| `deploy/prod` 外部 TLS | 已有 LB / CDN / WAF / Ingress 做 TLS 终止 | `TLS_MODE=external` | 容器内只跑 HTTP，HTTPS 由外层代理处理 |
| `deploy/prod` 内建 HTTPS | 单机公网直出、自己持有证书 | `TLS_MODE=redirect` | 证书路径写进 `.env`，容器内 Nginx 直接提供 HTTPS |

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
- 入口脚本：
  - `deploy/dev/scripts/infra.sh`
  - `deploy/dev/scripts/run-backend.sh`
  - `deploy/dev/scripts/build-app-base.sh`

### `prod/`

- 面向单机生产部署
- 重点是“文档清楚、.env 清楚、build.sh 清楚、compose 清楚”
- 入口文件：
  - `deploy/prod/.env.example`
  - `deploy/prod/build.sh`
  - `deploy/prod/docker-compose.yaml`

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
