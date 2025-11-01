# Composables

## 设计原则

1. **单一职责**：每个 Composable 只负责一个功能领域
2. **可复用**：逻辑可以在多个组件中复用
3. **可测试**：独立的函数，易于单元测试
4. **类型安全**：使用 TypeScript，提供完整的类型定义

## 已实现的 Composables

### useAppManager
负责应用管理的所有逻辑：
- 应用列表加载
- 应用切换
- 应用创建、更新、删除
- 从路由解析应用

**使用示例**：
```typescript
import { useAppManager } from '@/composables/useAppManager'

const {
  currentApp,
  appList,
  loading,
  loadAppList,
  switchApp,
  handleCreateApp,
  handleUpdateApp,
  handleDeleteApp
} = useAppManager()

// 加载应用列表
await loadAppList()

// 切换应用
await switchApp(app, true)
```

### useServiceTree
负责服务目录树管理：
- 服务树加载
- 节点查找和定位
- 目录创建

**使用示例**：
```typescript
import { useServiceTree } from '@/composables/useServiceTree'

const {
  serviceTree,
  loading,
  currentNode,
  loadServiceTree,
  locateNodeByRoute,
  handleCreateDirectory
} = useServiceTree()

// 加载服务树
await loadServiceTree(app)

// 定位节点
const node = locateNodeByRoute('/workspace/user/app/crm/ticket')
```

## 组件职责划分

### Composable（可复用逻辑）
- ✅ 数据管理（状态）
- ✅ API 调用
- ✅ 业务逻辑
- ✅ 数据转换

### Component（UI 展示）
- ✅ 模板渲染
- ✅ 用户交互
- ✅ 样式定义
- ✅ 事件处理

## 待实现的 Composables

### useFunctionRenderer
负责函数渲染相关逻辑：
- 函数详情加载
- 表单/表格渲染切换
- 执行结果处理

### useFormValidation
负责表单验证：
- 验证规则解析
- 表单验证执行
- 错误信息展示

### useTableOperations
负责表格操作：
- 搜索、排序、分页
- 行选择
- 批量操作

## 优势

### 相比旧版本（所有逻辑堆在组件里）

❌ **旧版本问题**：
- 单个组件 4800+ 行代码
- 逻辑混乱，难以维护
- 无法复用
- 难以测试

✅ **新版本优势**：
- 逻辑清晰，职责明确
- 代码复用率高
- 易于单元测试
- 便于扩展和维护

### 代码对比

**旧版本**（所有逻辑堆在组件里）：
```vue
<template>
  <!-- 4800 行模板 -->
</template>

<script setup>
// 1000+ 行逻辑
// 应用管理 + 服务树 + 表单 + 表格 + 验证 + ...
</script>
```

**新版本**（组件化）：
```vue
<template>
  <!-- 100 行模板 -->
</template>

<script setup>
import { useAppManager } from '@/composables/useAppManager'
import { useServiceTree } from '@/composables/useServiceTree'

// 30 行逻辑（只负责组合 Composables）
const { currentApp, switchApp } = useAppManager()
const { serviceTree, loadServiceTree } = useServiceTree()
</script>
```

## 最佳实践

1. **命名规范**：`use + 功能名称`
2. **返回值**：返回对象，包含状态和方法
3. **副作用**：明确标注有副作用的操作（如 API 调用）
4. **依赖注入**：通过参数传递依赖，不要直接导入
5. **错误处理**：统一的错误处理和提示

## 避免的坑

1. ❌ **不要在 Composable 中直接操作 DOM**
2. ❌ **不要在 Composable 中导入组件**
3. ❌ **不要让 Composable 依赖其他 Composable**（除非有充分理由）
4. ❌ **不要把所有逻辑都塞进一个 Composable**

## 总结

通过 Composables，我们实现了：
- ✅ **高内聚低耦合**
- ✅ **代码复用**
- ✅ **易于测试**
- ✅ **易于维护**

**永远记住**：避免屎山，从组件化开始！🚀

