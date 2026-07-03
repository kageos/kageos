# 贡献 Kageos

感谢你愿意改进 Kageos。本仓库当前采用 Business Source License 1.1 源码公开授权，不是 OSI open source。贡献前请阅读 [LICENSE](LICENSE) 和 [LICENSE_FAQ.md](LICENSE_FAQ.md)，确认你理解当前授权范围和未来 Apache-2.0 转换机制。

## 开始之前

- 较大的功能、架构调整或产品方向变化，请先和维护者讨论。
- 保持 PR 聚焦。不要把重构、格式化、依赖升级和产品行为变化混在一个 PR 中。
- 不要提交真实密钥、客户数据、生成的生产文件、license、`namespace/`、`local/` 或运行时数据目录。
- 在核心平台转为 Apache-2.0 之前，对外描述请使用“源码公开、可自托管”，不要称为 OSI open source。

## 贡献流程

1. Fork 仓库，并从最新 `main` 创建分支。
2. 按本文的本地开发流程启动 Kageos。
3. 进行小而清晰的修改。
4. 为 bug fix 或新功能补充测试，或在 PR 中说明验证方式。
5. 行为、部署、公开定位有变化时同步更新 README 或 docs。
6. 提交 Pull Request，并说明改了什么、为什么改、如何验证。

所有代码修改都应通过 Pull Request 合并。PR 需要通过 CI，并由维护者审核后合并。

## 分支模型

从 `main` 拉分支，推荐命名：

```text
feat/short-feature-name
fix/short-bug-name
docs/short-doc-change
chore/short-maintenance-change
```

提交 PR 前请同步最新 `main`，解决冲突，并确认没有提交本地运行时文件或私有数据。

## Commit 规范

推荐使用 Conventional Commits：

```text
feat: add scheduled agent notification tool
fix: prevent incomplete tool messages after interruption
docs: update production deployment guide
test: cover service tree bundle import
chore: refresh generated API docs
```

常用类型包括 `feat`、`fix`、`docs`、`refactor`、`test`、`chore`、`ci`。提交说明请尽量清晰，说明用户可见行为或维护目的。

## 本地开发

使用 `kagectl` 作为本地开发入口。仓库中仍保留底层 Compose 文件，但新贡献者不需要直接记住历史路径或手工配置环境变量。

只读运行可直接克隆上游仓库；准备提交 PR 时请先 fork：

```bash
git clone https://github.com/kageos/kageos.git
cd kageos
```

启动本地后端：

```bash
go run ./cmd/kagectl bootstrap --dev
```

这个命令会初始化本地 dev 工作区、启动 MySQL / NATS / MinIO、确保本地 user-app base image 存在，并以前台方式运行后端。默认优先使用 Podman；如果想用 Docker：

```bash
go run ./cmd/kagectl bootstrap --dev --engine docker
```

停止后端可按 `Ctrl-C`。停止本地基础设施：

```bash
go run ./cmd/kagectl down
```

另开终端启动前端：

```bash
cd web
npm install
npm run dev
```

启动后访问 `http://localhost:5173`。`kagectl bootstrap --dev` 会打印 `Kageos dev initialization summary`；使用用户名 `system` 和其中的 `Admin password` 登录。如果需要重新查看密码，可以执行 `go run ./cmd/kagectl init --dev`，或读取 `.kageos/dev/env/kageos.env` 中的 `SYSTEM_USER_PASSWORD`。

## 常用开发命令

```bash
go run ./cmd/kagectl status
go run ./cmd/kagectl doctor
go run ./cmd/kagectl verify
go run ./cmd/kagectl logs main
go run ./cmd/kagectl logs infra
go run ./cmd/kagectl down
```

## 本地验证

按修改范围运行对应检查。

后端：

```bash
bash scripts/test-core-go.sh
```

前端：

```bash
cd web
npm ci
npm run check:architecture
npm run lint
npm run type-check
npm run test:unit -- --run
npm run build
```

仓库治理：

```bash
bash scripts/check-sensitive-files.sh
bash scripts/check-sdk-boundaries.sh
bash scripts/check-doc-links.sh
git diff --check
```

安全依赖检查：

```bash
cd web
npm audit --omit=dev
```

```bash
go run golang.org/x/vuln/cmd/govulncheck@latest ./cmd/... ./core/... ./dto/... ./pkg/...
```

## PR 清单

- 修改有清晰原因，范围尽量小。
- 用户可见行为有测试，或 PR 中解释了验证方式。
- 行为、部署或公开定位变化时同步更新文档。
- 新日志不泄露 secret、token、license key、客户数据或私有工作区路径。
- 新依赖确有必要、仍在维护，并且与项目授权模型兼容。

## 行为准则和安全问题

参与社区请遵守 [CODE_OF_CONDUCT.zh-CN.md](CODE_OF_CONDUCT.zh-CN.md)。安全漏洞请按 [SECURITY.zh-CN.md](SECURITY.zh-CN.md) 私下报告，不要发公开 issue。

提交贡献即表示你同意将贡献按本仓库当前授权条款发布，并接受对应版本未来按 [LICENSE](LICENSE) 中的 Change License 机制转换为 Apache License 2.0，除非你和维护者另有书面约定。
