# Playwright MCP 接入指南

这份文档解决的是“让 AI 客户端直接操作本地开发中的 `AI-Agent-OS web` 页面”，不是替代仓库内的 Playwright Test。

推荐分层：

- `Playwright Test`：稳定回归、CI、可复现 smoke
- `playwright-mcp`：让 Codex、Cursor、Claude Code、VS Code 等客户端交互式打开页面、点页面、看控制台、拿 snapshot

官方参考：

- Playwright MCP: https://playwright.dev/docs/next/getting-started-mcp
- Playwright MCP Registry README: https://github.com/microsoft/playwright-mcp
- Playwright browser isolation: https://playwright.dev/docs/browser-contexts

## 1. 你们项目里的推荐模式

你们当前前端认证依赖：

- 浏览器内的 `localStorage.token`
- 请求时自动附加的 `X-Token`
- 401 时依赖 `refresh_token`

因此，MCP 这边不要只想着“给请求头塞 token”，而要准备完整浏览器状态。

基于这一点，仓库里放了两套 MCP 配置：

- `tests/e2e/playwright.mcp.persistent.config.json`
  - 适合本地长期开发
  - 使用持久化 `userDataDir`
  - 第一次登录后，后续会话可复用浏览器 profile
- `tests/e2e/playwright.mcp.isolated.config.json`
  - 适合和当前 E2E 登录态联动
  - 使用 `storageState=tests/e2e/.auth/user.json`
  - 更可控，更接近“测试会话”

## 2. 前置条件

先启动后端和前端：

```bash
bash deploy/dev/scripts/run-backend.sh
cd web
npm run dev -- --host 127.0.0.1
```

如果本机还没装过浏览器运行时：

```bash
cd web
npm run playwright:install
```

## 3. 推荐工作流

### 方案 A：本地长期开发，优先用 persistent

启动 MCP：

```bash
cd web
npm run playwright:mcp
```

特点：

- 浏览器状态保存在 `tests/e2e/.mcp-profile/`
- 第一次你可以手工登录，或者让 AI 帮你登录
- 后续 MCP 会话继续复用这份 profile

适合：

- 日常联调
- 长时间开着让 AI 帮你排查 UI 问题
- 不想每次都重新准备登录态

### 方案 B：和测试态对齐，优先用 isolated

先生成认证状态文件：

```bash
cd web
E2E_USERNAME=system \
E2E_PASSWORD=your-password \
npm run test:e2e:smoke
```

如果你已经有 token，也可以直接跑：

```bash
cd web
E2E_TOKEN=your-access-token \
E2E_REFRESH_TOKEN=your-refresh-token \
E2E_USERNAME=system \
npm run test:e2e:smoke
```

这一步会刷新 `tests/e2e/.auth/user.json`。

然后启动 MCP：

```bash
cd web
npm run playwright:mcp:isolated
```

特点：

- 每次 MCP 会话从 `storageState` 起步
- 更可控，适合复现和回归
- 不会长期污染 profile

适合：

- 让 AI 做稳定问题复现
- 对齐 Playwright Test 行为
- 需要快速重置状态

## 4. HTTP 模式

如果你的 MCP 客户端更适合连一个长期运行的 HTTP endpoint：

```bash
cd web
npm run playwright:mcp:http
```

默认会在 `http://localhost:8931/mcp` 提供服务。

适合：

- IDE worker 进程
- 一个本地 Playwright MCP 服务给多个客户端复用

## 5. 客户端配置示例

### Codex

官方支持直接添加：

```bash
codex mcp add playwright npx "@playwright/mcp@latest"
```

如果你要显式带本仓库配置文件，更推荐在 `~/.codex/config.toml` 里写成：

```toml
[mcp_servers.playwright]
command = "npx"
args = ["-y", "@playwright/mcp@latest", "--config", "/绝对路径/ai-agent-os/web/tests/e2e/playwright.mcp.persistent.config.json"]
```

如果你更想走隔离态，就把最后那个 config 路径替换成：

```text
/绝对路径/ai-agent-os/web/tests/e2e/playwright.mcp.isolated.config.json
```

### Cursor / VS Code / Claude Desktop

标准 MCP 配置都可以复用同一个命令，只需要把 config 路径换成你本机的绝对路径：

```json
{
  "mcpServers": {
    "playwright": {
      "command": "npx",
      "args": [
        "-y",
        "@playwright/mcp@latest",
        "--config",
        "/绝对路径/ai-agent-os/web/tests/e2e/playwright.mcp.persistent.config.json"
      ]
    }
  }
}
```

### Claude Code

```bash
claude mcp add playwright -- npx -y @playwright/mcp@latest --config /绝对路径/ai-agent-os/web/tests/e2e/playwright.mcp.persistent.config.json
```

## 6. 为什么仓库里要同时保留 Playwright Test 和 MCP

因为它们解决的问题不同：

- `Playwright Test` 保证“这条链路可回归”
- `playwright-mcp` 保证“AI 能探索页面并直接操作”

你们项目不是普通站点，而是：

- 有登录态
- 有服务树
- 有动态 `Form / Table / Chart / Workstation`

所以不能只靠浏览器 MCP，也不能只靠测试 runner。

## 7. 当前仓库内已经配好的配套点

以下组件已经补了 `data-testid`，方便 MCP 和 Playwright 共用定位：

- 登录页
- 工作空间壳层
- 服务树面板
- 函数渲染区域
- 表单页
- 表格页
- 工作台对话区

如果后面要继续增强，建议下一批优先补：

- `WorkspaceHeader`
- `WorkspaceSidebarSessionsPanel`
- `MiniWorkstation`
- 表格操作下拉菜单
- 关键弹窗的确认按钮

## 8. 你们项目里的实际建议

日常开发默认走：

```bash
npm run playwright:mcp
```

需要稳定复现问题时走：

```bash
npm run test:e2e:smoke
npm run playwright:mcp:isolated
```

这样分工最清晰：

- persistent 负责“方便”
- isolated 负责“稳定”

不要把这两件事混成一条链路。
