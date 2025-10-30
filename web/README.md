# AI Agent OS 前端项目

基于 Vue 3 + TypeScript + Element Plus 构建的现代化前端应用。

## 项目特性

- ⚡️ Vue 3 + Vite - 快速的开发体验
- 🛠️ TypeScript - 完整的类型安全
- 🎨 Element Plus - 企业级 UI 组件库
- 📦 Pinia - 状态管理
- 🚀 Vue Router - 路由管理
- 🧪 Vitest - 单元测试
- 📖 ESLint - 代码质量保证

## 技术栈

- **前端框架**: Vue 3.5+
- **构建工具**: Vite
- **开发语言**: TypeScript
- **UI 组件库**: Element Plus
- **状态管理**: Pinia
- **路由管理**: Vue Router
- **HTTP 请求**: Axios (待安装)
- **样式方案**: CSS3 + Element Plus
- **代码规范**: ESLint + Prettier

## 项目结构

```
web/
├── public/                 # 静态资源
├── src/
│   ├── assets/            # 资源文件
│   │   ├── main.css      # 全局样式
│   │   └── logo.svg      # 项目logo
│   ├── components/       # 公共组件
│   │   └── HelloWorld.vue
│   ├── views/           # 页面组件
│   │   ├── AboutView.vue
│   │   ├── Demo/        # 演示页面
│   │   │   └── index.vue
│   │   └── HomeView.vue
│   ├── router/          # 路由配置
│   │   └── index.ts
│   ├── stores/          # 状态管理
│   │   └── counter.ts
│   ├── App.vue          # 根组件
│   ├── main.ts          # 应用入口
│   └── env.d.ts         # 类型声明
├── package.json         # 项目配置
├── tsconfig.json        # TypeScript 配置
├── vite.config.ts       # Vite 配置
└── README.md           # 项目说明
```

## 快速开始

### 环境要求

- Node.js >= 18.0.0
- npm >= 8.0.0

### 安装依赖

```bash
npm install
```

### 启动开发服务器

```bash
npm run dev
```

### 构建生产版本

```bash
npm run build
```

### 预览生产版本

```bash
npm run preview
```

### 运行测试

```bash
npm run test:unit
```

## 页面说明

### 首页 (/)
- 展示 AI Agent OS 的核心特性
- 提供快速导航和技术栈介绍

### 演示页面 (/demo)
- Element Plus 组件演示
- 包含表单、表格、分页、对话框等常用组件
- 实际交互示例

### 关于页面 (/about)
- 项目介绍信息

## Element Plus 配置

项目已配置 Element Plus 自动导入：

- 自动导入组件：`unplugin-vue-components`
- 自动导入 API：`unplugin-auto-import`
- 样式文件：在 `main.ts` 中全局引入

### 使用示例

```vue
<template>
  <el-button type="primary">主要按钮</el-button>
  <el-input v-model="input" placeholder="请输入内容" />
  <el-table :data="tableData">
    <el-table-column prop="name" label="姓名" />
  </el-table>
</template>
```

## 开发规范

### 命名规范

- 组件文件：PascalCase（如 `UserList.vue`）
- 页面文件：PascalCase（如 `HomeView.vue`）
- 变量和方法：camelCase（如 `getUserInfo`）
- 常量：SCREAMING_SNAKE_CASE（如 `API_BASE_URL`）

### 代码规范

- 使用 TypeScript 进行类型检查
- 遵循 ESLint 配置的代码规范
- 组件使用 Composition API + `<script setup>` 语法

### Git 提交规范

- `feat`: 新功能
- `fix`: 修复bug
- `docs`: 文档更新
- `style`: 代码格式调整
- `refactor`: 代码重构
- `test`: 测试相关
- `chore`: 构建过程或辅助工具的变动

## 待实现功能

- [ ] API 接口集成
- [ ] 用户认证系统
- [ ] 代理管理界面
- [ ] 服务目录功能
- [ ] 实时监控面板
- [ ] 响应式设计优化
- [ ] 国际化支持
- [ ] 主题切换功能

## 部署说明

项目支持多种部署方式：

### Docker 部署

```dockerfile
FROM node:18-alpine as build-stage
WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production
COPY . .
RUN npm run build

FROM nginx:stable-alpine as production-stage
COPY --from=build-stage /app/dist /usr/share/nginx/html
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
```

### 静态文件部署

构建完成后，`dist` 目录可直接部署到任何静态文件服务器。

## 联系方式

如有问题或建议，请提交 Issue 或联系开发团队。
