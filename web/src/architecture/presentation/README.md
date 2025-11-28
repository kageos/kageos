# Presentation Layer (UI 组件层)

## 职责

- 纯 UI 展示，不包含业务逻辑
- 通过事件与 Application Layer 通信
- 从 StateManager 获取状态并渲染

## 目录结构

- `views/`：页面级视图组件（WorkspaceView、FormView、TableView）
- `widgets/`：Widget 组件（从 `core/widgets-v2/components/` 迁移）
- `components/`：通用 UI 组件（可选）

## 特点

- 组件代码量小（< 300 行/个）
- 只负责展示和用户交互
- 不直接调用 API，通过事件通信

## 使用示例

```vue
<template>
  <WorkspaceView />
</template>

<script setup lang="ts">
import WorkspaceView from '@/architecture/presentation/views/WorkspaceView.vue'
</script>
```

