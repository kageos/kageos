# 源码导览与贡献参考

## 目录

- 开始前
- 仓库地图
- 修改流程
- 验证矩阵
- PR 交付

## 开始前

先阅读目标仓库当前版本的：

- `CONTRIBUTING.md`
- `LICENSE` 和 `LICENSE_FAQ.md`
- `docs/current-architecture.md`
- `docs/kageos-blueprint.md`

核心仓库当前采用 BSL 1.1，并根据仓库许可说明按版本转换许可证。对外描述使用“source-available and self-hostable”，不要误称当前核心仓库为开源许可证项目。

## 仓库地图

| 路径 | 主要职责 |
|---|---|
| `cmd/kagectl` | 本地、生产和 AIO 生命周期入口 |
| `core` | 平台各服务和主进程 |
| `pkg` | 平台共享实现 |
| `dto` | 跨模块数据契约 |
| `web` | Vue 3 前端 |
| `deploy` | 开发、生产、镜像和安全部署材料 |
| `docs` | 架构、产品、SOP 和治理文档 |
| `core/app-server/system-seed` | 内置 capability bundle 种子目录 |

工作空间应用应该依赖独立的 `github.com/kageos/kageos-sdk`，不应依赖主仓内部实现。`namespace` 和 `.kageos` 属于运行态数据，不是普通平台源码。

定位功能时先搜索路由、DTO、服务名和测试，再沿调用链阅读。架构描述以当前源码和架构文档相互印证。

## 修改流程

1. 检查 `git status --short --branch` 和 remote。
2. 若准备提交贡献，从最新 `main` 创建 `feat/...`、`fix/...`、`docs/...` 或 `chore/...` 分支。
3. 写出最小复现或明确验收条件。
4. 实施聚焦修改，同时补充测试。
5. 行为、部署或公开定位变化时更新 README/docs。
6. 审查 diff，排除 `.kageos/`、`namespace/`、`local/`、密钥、客户数据、日志和生成生产文件。

大功能在编码前建议先与维护者讨论。不要把重构、格式化扫荡、依赖升级和行为变化混在同一个 PR。

## 验证矩阵

按修改范围运行，未运行的检查必须说明原因。

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

依赖安全检查按任务风险选择：

```bash
cd web
npm audit --omit=dev
```

```bash
go run golang.org/x/vuln/cmd/govulncheck@latest ./cmd/... ./core/... ./dto/... ./pkg/...
```

修复局部 Go 包时可先运行目标包测试加速反馈，交付前再运行仓库要求的对应完整检查。

## PR 交付

优先使用 Conventional Commits，例如：

```text
feat: add contributor startup diagnostics
fix: preserve dev state when bootstrap retries
docs: clarify local Docker workflow
```

PR 说明至少包括：

- 改了什么；
- 为什么改；
- 如何复现或验证；
- 已运行和未运行的检查；
- UI 变化截图或部署影响（如适用）。

默认只准备内容。commit、push、创建 PR 和对外评论都需要用户明确要求。
