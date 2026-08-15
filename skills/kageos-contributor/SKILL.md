---
name: kageos-contributor
description: 帮助用户把 kageos 平台源码拉到本地电脑，检查并在明确同意后补齐 Git、Go、Node.js、Podman 或 Docker 等依赖，通过 kagectl 启动前后端、验证健康状态、打开本地页面并排查启动问题；启动成功后继续源码导览、问题复现、分支修改、测试和 PR 准备。用户说想在本地打开、运行、调试、阅读或贡献 kageos 源码，或询问 kageos 为什么启动失败时使用。不要用于开发 kageos 工作空间目录应用；那类请求使用 kageos-developer。
---

# kageos contributor

先让用户在本地成功打开 kageos，再进入源码阅读和贡献。名称 `kageos` 只表示产品名称，不要扩写或解释。

## 判断目标

将请求归入一个或多个阶段：

1. `local_start`：环境检查、拉取源码、启动、登录和打开页面。
2. `troubleshoot`：诊断已有本地环境，恢复到可运行状态。
3. `source_tour`：只读定位架构、模块和调用链。
4. `contribute`：复现问题、创建分支、修改、验证和准备 PR。

默认从用户尚未完成的最早阶段开始。用户只要求查看源码时保持只读；用户只要求启动时，不擅自修改产品代码或创建 PR。

## 本地启动闭环

### 1. 发现本地状态

- 先判断当前目录或用户给定目录是否已经是 kageos 源码仓库。使用 `.kageos-root`、`go.mod` 和 `cmd/kagectl` 共同确认，不靠目录名猜测。
- 未指定仓库位置且本地没有仓库时，提出一个明确目标路径。不要把仓库克隆进已有内容的非空目录。
- 运行本 skill 的 `scripts/contributor_doctor.sh`。脚本路径使用 skill 根目录的绝对路径，不要假定当前工作目录。
- 需要安装和启动细节时，完整读取 `references/local-startup.md`。

### 2. 安装前确认

只读检测、克隆公开仓库和读取源码可以直接进行。安装系统软件、初始化新的 Podman machine、修改 shell 配置或要求管理员权限前，向用户说明准确动作和影响并获得确认。

优先复用已经健康运行的 Docker 或 Podman。用户已有 Docker 时，不要求额外安装 Podman。两者都没有时，根据操作系统和可用包管理器给出最短方案，不静默安装。

Podman 安全门禁：

- `podman info` 失败时先查看 `podman machine list`、连接状态和磁盘空间。
- 已存在 machine 或疑似存在数据时，禁止直接执行 `podman machine init`、`podman machine rm` 或删除 raw 磁盘。
- 新机器上确认没有任何 Podman machine 后，只有获得用户确认才可以初始化一个新的 machine。
- 任何清数据、删 volume 或重建 machine 的动作都视为破坏性操作，必须单独确认。

### 3. 获取源码

- 只想查看或运行：克隆 `https://github.com/kageos/kageos.git`。
- 明确准备贡献：优先确认用户的 GitHub fork；若 GitHub CLI 已鉴权，可在用户同意后 fork 并克隆。否则先克隆 upstream，成功启动后再配置 fork。
- 克隆后先阅读仓库里的 `README.md`、`CONTRIBUTING.md`、`LICENSE` 和 `LICENSE_FAQ.md`。说明核心仓库当前是 source-available、自托管，采用 BSL 1.1，并按仓库许可说明转换许可证。
- 保护已有工作树。修改前检查 `git status --short --branch`，不覆盖用户改动。

### 4. 启动并保持进程可见

在仓库根目录运行官方入口，不直接操作底层 Compose：

```bash
go run ./cmd/kagectl bootstrap --dev
```

若已选择 Docker：

```bash
go run ./cmd/kagectl bootstrap --dev --engine docker
```

让后端运行在一个可继续读取输出的终端会话中。确认后端没有立即退出后，在第二个终端会话运行：

```bash
cd web
npm install
npm run dev
```

不要用无法追踪的裸后台进程掩盖错误。长时间下载或构建期间持续向用户报告当前阶段。

### 5. 验证并交付入口

依次执行或读取：

```bash
go run ./cmd/kagectl doctor
go run ./cmd/kagectl status
go run ./cmd/kagectl verify
```

确认 `http://localhost:5173` 可访问，再使用可用的浏览器工具打开。告诉用户：

- 本地地址；
- 用户名 `system`；
- 密码来自启动摘要，或可通过 `go run ./cmd/kagectl init --dev` 重新显示；
- 当前源码路径；
- 后端、前端分别在哪个终端会话运行；
- 如何停止：先停止两个前台进程，再运行 `go run ./cmd/kagectl down`。

不要在对话、日志摘要、提交或制品中复述 `.kageos/dev/env/kageos.env` 的密码和密钥。

## 启动排障

按固定顺序排查，不先碰底层 Compose：

1. 读取最早失败的终端输出。
2. 运行 `kagectl doctor`。
3. 运行 `kagectl status`。
4. 运行 `kagectl verify`。
5. 按需读取 `kagectl logs main` 或 `kagectl logs infra`。
6. 检查 Go、Node、容器引擎、磁盘空间和端口占用。

修复后从失败步骤继续，不重复生成密钥或清理已有数据。涉及 Podman machine 异常时遵守上面的安全门禁。

## 源码导览和贡献

启动成功后，询问用户要查看哪个功能或贡献哪个问题，不强迫进入贡献流程。

源码导览和代码修改前完整读取 `references/contribution-flow.md`。优先从仓库自己的 `docs/current-architecture.md`、`docs/kageos-blueprint.md` 和相关模块测试建立证据，不凭记忆描述架构。

贡献时：

- 从最新 `main` 建立聚焦分支；大功能先建议与维护者讨论。
- 先复现，再做最小修改；不混入格式化扫荡、无关重构或依赖升级。
- 根据修改范围运行验证矩阵，并检查敏感文件、SDK 边界和文档链接。
- 使用 Conventional Commits。
- 可以准备 commit 和 PR 内容；只有用户明确要求时才 commit、push 或创建 PR。
- 不提交 `.kageos/`、`namespace/`、`local/`、真实密钥、客户数据、日志或生成的生产文件。

## 完成标准

`local_start` 只有在以下事实全部成立后才算完成：

- 源码存在于明确路径；
- 一个容器引擎和 Compose 可用；
- `kagectl verify` 通过，或已准确说明仍失败的检查；
- 前端地址可访问；
- 用户知道登录和停止方法。

贡献任务还必须返回：修改范围、验证结果、未运行的检查及原因、工作树状态，以及 commit/PR 是否已经创建。
