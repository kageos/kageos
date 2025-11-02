# 新版渲染引擎

## 目录结构

```
core/
├── types/              # 类型定义
│   ├── field.ts       # 字段相关类型
│   └── widget.ts      # Widget 相关类型
├── widgets/           # Widget 组件
│   ├── BaseWidget.ts  # 基类
│   └── InputWidget.ts # 输入框组件
├── managers/          # 管理器
│   └── ReactiveFormDataManager.ts  # 表单数据管理器
├── factories/         # 工厂
│   └── WidgetFactory.ts  # Widget 工厂
└── renderers/         # 渲染器
    └── FormRenderer.vue  # 表单渲染器（新架构）
```

## 已实现功能

### 1. 基础架构
- ✅ `BaseWidget` 基类：提供快照、渲染等基础功能
- ✅ `WidgetFactory` 工厂：根据类型动态创建 Widget
- ✅ `ReactiveFormDataManager` 管理器：管理表单数据

### 2. 组件
- ✅ `InputWidget`：输入框组件

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
import FormRenderer from '@/core/renderers/FormRenderer.vue'
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

1. **OOP 设计**：使用类和继承，代码清晰易维护
2. **平铺结构**：所有 Widget 平铺存储，通过 field_path 标识
3. **工厂模式**：动态创建组件，易扩展
4. **快照机制**：支持表单状态的保存和恢复
5. **响应式管理**：基于 Vue 3 响应式系统

## 注意事项

1. 当前只实现了 `InputWidget`，其他组件待开发
2. 表单验证暂未实现
3. 嵌套结构（List、Struct）暂未实现
4. 回调系统暂未实现

## 调试

### 控制台输出

所有 Widget 和 Manager 都有详细的控制台日志，格式如下：

```
[BaseWidget] 创建 Widget: username, depth: 0
[ReactiveFormDataManager] 初始化字段: username
[FormRenderer] 注册 Widget: username
```

### 调试按钮

点击"调试输出"按钮可以查看：
- 函数详情
- 字段列表
- 所有字段路径
- 提交数据
- 已注册的 Widget
- 已注册的 Widget 类型

