# Hub Frontend

Hub 应用市场前端项目。

## 技术栈

- **Vue 3** - 前端框架
- **TypeScript** - 类型支持
- **Vite** - 构建工具
- **Element Plus** - UI 组件库
- **Vue Router** - 路由管理
- **Pinia** - 状态管理
- **Axios** - HTTP 客户端

## 项目结构

```
hub/frontend/
├── src/
│   ├── api/          # API 客户端
│   ├── assets/       # 静态资源
│   ├── config/       # 配置文件
│   ├── router/       # 路由配置
│   ├── stores/       # 状态管理
│   ├── utils/        # 工具函数
│   ├── views/        # 页面组件
│   ├── App.vue       # 根组件
│   └── main.ts       # 入口文件
├── index.html        # HTML 模板
├── package.json      # 项目配置
├── vite.config.ts    # Vite 配置
└── tsconfig.json     # TypeScript 配置
```

## 开发

```bash
# 安装依赖
npm install

# 启动开发服务器
npm run dev

# 构建生产版本
npm run build

# 预览生产构建
npm run preview
```

## 环境变量

创建 `.env` 文件：

```bash
# Hub API 基础地址（可选，默认通过代理）
VITE_HUB_API_BASE_URL=/hub/api/v1

# OS 登录页地址（用于 401 跳转）
VITE_OS_LOGIN_URL=http://localhost:5173/login
```

## 与 OS 的集成

Hub 和 OS 共享用户系统：
- 使用相同的 JWT Token
- 401 错误时跳转到 OS 登录页
- 通过网关代理 API 请求

