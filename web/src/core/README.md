# Core 核心模块

> **注意**：Widget 组件、工厂和渲染器已迁移到新架构 `web/src/architecture/` 目录下。
> 旧架构代码（widgets-v2、factories-v2、renderers-v2）已删除。

## 目录结构

```
core/
├── types/              # 类型定义（共享类型，新架构也使用）
│   ├── field.ts       # 字段相关类型
│   └── widget.ts      # Widget 相关类型
├── constants/          # 常量定义（共享常量）
│   ├── field.ts       # 字段常量
│   └── widget.ts      # Widget 常量
├── managers/          # 管理器（共享管理器）
│   └── ReactiveFormDataManager.ts  # 表单数据管理器
├── stores-v2/         # Pinia Stores（共享状态管理）
│   ├── formData.ts    # 表单数据 Store
│   └── responseData.ts # 响应数据 Store
├── utils/             # 工具函数（共享工具）
│   ├── logger.ts      # 日志工具
│   └── validationUtils.ts # 验证工具
└── validation/        # 验证系统（共享验证）
    └── ValidationEngine.ts # 验证引擎
```

## 已实现功能

### 1. 基础架构
- ✅ `WidgetComponentFactory` 工厂：根据类型动态获取 Vue 组件
- ✅ `ReactiveFormDataManager` 管理器：管理表单数据
- ✅ 所有 Widget 已迁移到 Vue 组件版本（widgets-v2）

### 2. Widget 组件（widgets-v2）
- ✅ `InputWidget`：文本输入
- ✅ `NumberWidget`：整数输入
- ✅ `FloatWidget`：浮点数输入
- ✅ `TextAreaWidget`：多行文本
- ✅ `SelectWidget`：下拉选择（单选）
- ✅ `MultiSelectWidget`：下拉选择（多选）
- ✅ `CheckboxWidget`：复选框
- ✅ `RadioWidget`：单选框
- ✅ `SwitchWidget`：开关
- ✅ `TimestampWidget`：时间戳
- ✅ `FilesWidget`：文件上传
- ✅ `UserWidget`：用户选择
- ✅ `TableWidget`：表格（嵌套）
- ✅ `FormWidget`：表单（嵌套）

### 3. 渲染器
- ✅ `FormRenderer`：表单渲染器，支持嵌套结构、回调、聚合等完整功能

## 测试

### 访问测试页面

启动开发服务器后，访问：

```
http://localhost:5173/test/form-renderer
```

### 测试功能

1. **基础渲染**：测试 InputWidget 的渲染
2. **数据绑定**：输入内容，查看数据变化
3. **表单提交**：点击提交按钮，查看控制台输出
4. **调试模式**：点击"调试输出"按钮，查看详细信息
5. **切换数据**：点击"切换测试数据"按钮，测试不同表单

## 下一步计划

### Phase 1：基础组件（当前）
- ✅ InputWidget
- ⬜ TextAreaWidget
- ⬜ NumberWidget
- ⬜ SelectWidget
- ⬜ MultiSelectWidget

### Phase 2：容器组件
- ✅ TableWidget
- ⬜ StructWidget

### Phase 3：高级功能
- ⬜ 字段验证
- ⬜ 条件渲染
- ⬜ 回调系统
- ⬜ 聚合统计

### Phase 4：优化
- ⬜ 性能优化
- ⬜ 错误处理
- ⬜ 快照系统

## 使用示例

```vue
<template>
  <FormRenderer :function-detail="functionDetail" />
</template>

<script setup lang="ts">
import FormRenderer from '@/core/renderers-v2/FormRenderer.vue'
import type { FunctionDetail } from '@/core/types/field'

const functionDetail: FunctionDetail = {
  code: 'test_form',
  name: '测试表单',
  method: 'POST',
  router: '/test/form',
  template_type: 'form',
  request: [
    {
      code: 'username',
      name: '用户名',
      validation: 'required',
      widget: { type: 'input' }
    }
  ],
  response: []
}
</script>
```

## 架构特点

1. **Vue 组件化**：所有 Widget 都是 Vue 3 组件，使用 Composition API
2. **工厂模式**：通过 `WidgetComponentFactory` 动态获取组件
3. **响应式管理**：基于 Vue 3 响应式系统和 Pinia Store
4. **类型安全**：完整的 TypeScript 类型定义
5. **可扩展性**：新增 Widget 只需在 `widgets-v2/components/` 中添加组件并注册到工厂

## 使用场景

### 1. 表单渲染（FormRenderer）
- 使用 `widgets-v2/components/*.vue` 组件
- 支持编辑模式（edit）、响应模式（response）

### 2. 表格渲染（TableRenderer）
- 使用 `widgets-v2/components/*.vue` 组件
- 支持表格单元格模式（table-cell）、详情模式（detail）

### 3. 搜索输入（SearchInput）
- 使用 `widgets-v2/components/*.vue` 组件
- 支持搜索模式（search）

## 迁移说明

旧版本的 `widgets/` 目录和 `factories/WidgetBuilder.ts`、`factories/WidgetFactory.ts` 已完全移除。

所有功能已迁移到 `widgets-v2/` 和 `factories-v2/`。

