# Presentation Layer (UI 组件层)

## 职责

- 纯 UI 展示，不包含业务逻辑
- 通过事件与 Application Layer 通信
- 从 StateManager 获取状态并渲染

## 目录结构

- `views/`：页面级视图组件（WorkspaceView、FormView、TableView）
- `features/`：认证、用户、组织、消息、Agent 配置等业务入口页面
- `shared/`：跨业务复用展示组件、富文本编辑器和展示类型
- `widgets/`：Widget 组件；通用字段值与校验规则放在 `architecture/domain/utils/`
  - `registry/`：Widget 组件工厂与注册入口
  - `plugins/`：Widget 插件注册与扩展编排
- `components/`：主链路 UI 组件
- `router/`：Vue Router 路由表、页面装配与全局守卫
- `assets/` / `styles/`：展示层资源与全局样式

## 特点

- 只负责展示和用户交互
- 不直接调用 API，通过事件通信
- 业务入口、共享组件和主链路组件都在同一个 presentation 边界内

## 使用示例

```vue
<template>
  <WorkspaceView />
</template>

<script setup lang="ts">
import WorkspaceView from '@/architecture/presentation/views/WorkspaceView.vue'
</script>
```
