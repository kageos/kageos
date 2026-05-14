# Playwright E2E

这套基建先解决两件事：

- 给 `web/` 提供稳定的 Chromium smoke / E2E 入口
- 让浏览器自动化能复用登录态，而不是每条用例都手工登录

## 启动依赖

先在仓库根目录启动后端，再在 `web/` 启前端：

```bash
APP_ENV=dev go run ./core/cmd/main
cd web
npm run dev -- --host 127.0.0.1
```

日常本地开发也可以直接用 GoLand 启动 `core/cmd/main/main.go`，环境变量设置为 `APP_ENV=dev`。

首次使用需要安装 Playwright 浏览器：

```bash
cd web
npm run playwright:install
```

## 登录模式

全局 setup 支持两种模式，二选一。

如果只是验证无需登录的公共页面，可跳过登录态准备：

```bash
cd web
npm run test:e2e:login
```

### 1. 真实登录

```bash
cd web
E2E_USERNAME=system \
E2E_PASSWORD=your-password \
npm run test:e2e
```

### 2. 直接注入已有 token

```bash
cd web
E2E_TOKEN=your-access-token \
E2E_REFRESH_TOKEN=your-refresh-token \
E2E_USERNAME=system \
npm run test:e2e
```

如果你已经有完整用户 JSON，也可以直接传：

```bash
E2E_USER_JSON='{"username":"system"}'
```

## 常用环境变量

- `PLAYWRIGHT_BASE_URL`: 默认 `http://127.0.0.1:5173`
- `PLAYWRIGHT_LOGIN_TIMEOUT_MS`: 登录与跳转超时，默认 `30000`
- `E2E_USERNAME`: UI 登录用户名，或 token 注入时用于构造最小用户信息
- `E2E_PASSWORD`: UI 登录密码
- `E2E_TOKEN`: 直接注入 `localStorage.token`
- `E2E_REFRESH_TOKEN`: 直接注入 `localStorage.refresh_token`
- `E2E_USER_JSON`: 直接注入 `localStorage.user`

## 目录说明

- `global.setup.ts`: 统一准备登录态并写入 `tests/e2e/.auth/user.json`
- `login-page.spec.ts`: 无登录态公共登录页 smoke
- `smoke.spec.ts`: 最小工作空间壳层 smoke
- `workspace-navigation.spec.ts`: 工作区选择器与创建工作区弹窗 smoke
- `service-tree-actions.spec.ts`: 服务树根目录动作与创建目录弹窗 smoke
- `workstation-open.spec.ts`: 服务树打开迷你工作台 smoke
- `workstation-composer.spec.ts`: 迷你工作台输入后发送按钮激活 smoke

## 常用命令

```bash
npm run test:e2e:login
npm run test:e2e:smoke
npm run test:e2e:workspace-nav
npm run test:e2e:service-tree
npm run test:e2e:workstation
npm run test:e2e:workstation-compose
```

## 和 MCP 的关系

这套 Playwright Test 基建负责“可回归、可复现”。  
`playwright-mcp` 负责“AI 交互式看页面和点页面”。

建议先用这里的账号 / token 约定把开发环境跑通，再把同一套环境变量或测试账号复用到 MCP 客户端。

如果本机没有 docker 或暂时不跑后端，也可以先用 `npm run test:e2e:login` 验证：

- Playwright 是否能自动拉起 Vite
- 浏览器自动化链路是否正常
- `data-testid` 和基本页面壳层是否可见

仓库内的 MCP 接入样板和多客户端配置示例见：

- [`web/docs/Playwright-MCP接入指南.md`](../docs/Playwright-MCP接入指南.md)
