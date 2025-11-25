# Widget 系统重构方案：从 widgets/ v1 到 widgets-v2/

## 🎯 重构目标

**不是迁移，而是重构**：按照 widgets-v2 的设计思路和风格重新实现，而不是简单复制代码。

### 核心原则

1. **按照 v2 的思路重新实现**：不是复制代码，而是理解 v2 的设计理念后重新实现
2. **保证可用性和维护性**：重构的目标是提高代码质量和可维护性
3. **v1 不一定是最合适的**：v1 的实现可能有历史包袱，需要重新审视和优化

## 📊 v1 vs v2 设计差异分析

### v1 的设计（基于类）

**特点**：
- 基于 TypeScript 类的继承体系
- 每个 Widget 有多个方法：`renderTableCell()`, `renderForDetail()`, `renderSearchInput()`
- 使用 `h()` 函数式渲染，返回 VNode 或字符串
- 需要创建临时 Widget 实例

**问题**：
- ❌ 不符合 Vue 3 最佳实践（基于类，不是 Composition API）
- ❌ 代码分散在多个方法中，难以维护
- ❌ 需要创建实例，增加复杂度
- ❌ 方法调用方式，不够直观

**示例**：
```typescript
// v1: 创建实例，调用方法
const tempWidget = WidgetBuilder.createTemporary({ field, value })
const result = tempWidget.renderTableCell(value, userInfoMap)
// 返回：string | VNode
```

### v2 的设计（基于 Vue 组件）

**特点**：
- 基于 Vue 3 Composition API
- 统一的 Props 接口，通过 `mode` prop 区分场景
- 使用模板语法，更符合 Vue 3 最佳实践
- 使用 Pinia Store 管理状态
- 使用 composables 提取共享逻辑

**优势**：
- ✅ 符合 Vue 3 最佳实践
- ✅ 代码集中在组件中，易于维护
- ✅ 直接使用组件，无需创建实例
- ✅ 模板语法，更直观

**示例**：
```vue
<!-- v2: 直接使用组件 -->
<component 
  :is="widgetComponent"
  :field="field"
  :value="value"
  mode="table-cell"
  :user-info-map="userInfoMap"
/>
```

## 🔍 重构点分析

### 1. TableRenderer.vue - renderTableCell

#### v1 的实现方式
```typescript
// 创建临时 Widget 实例
const tempWidget = WidgetBuilder.createTemporary({
  field: coreField,
  value: value
})

// 调用方法
const result = tempWidget.renderTableCell(value, userInfoMap.value)

// 返回：string | VNode
return {
  content: result,
  isString: typeof result === 'string',
  isVNode: !isString && isVNode(result)
}
```

#### v2 的重构思路

**问题分析**：
- v1 需要创建实例，调用方法，返回 VNode 或字符串
- v2 是 Vue 组件，需要渲染为 VNode

**重构方案**：
```typescript
// 使用 widgetComponentFactory 获取组件
const WidgetComponent = widgetComponentFactory.getRequestComponent(
  field.widget?.type || 'input'
)

// 使用 h() 渲染组件为 VNode
const vnode = h(WidgetComponent, {
  field: field,
  value: value,
  'model-value': value,
  'field-path': field.code,
  mode: 'table-cell',
  'user-info-map': userInfoMap.value
})

// 统一返回 VNode（不再需要区分字符串和 VNode）
return {
  content: vnode,
  isString: false,
  isVNode: true
}
```

**改进点**：
- ✅ 直接使用组件，无需创建实例
- ✅ 统一的 Props 接口，更清晰
- ✅ 符合 Vue 3 最佳实践

### 2. TableRenderer.vue - renderDetailField

#### v1 的实现方式
```typescript
const widget = WidgetBuilder.createTemporary({
  field: field,
  value: value
})

const result = widget.renderForDetail(value, context)

// 如果返回字符串，包装成 VNode
if (typeof result === 'string') {
  return h('span', result)
}
return result
```

#### v2 的重构思路

**问题分析**：
- v1 需要处理字符串和 VNode 两种情况
- v2 组件统一返回 VNode

**重构方案**：
```typescript
const WidgetComponent = widgetComponentFactory.getRequestComponent(
  field.widget?.type || 'input'
)

// 直接渲染组件，统一返回 VNode
return h(WidgetComponent, {
  field: field,
  value: value,
  'model-value': value,
  'field-path': field.code,
  mode: 'detail',
  'user-info-map': userInfoMap.value
})
```

**改进点**：
- ✅ 统一返回 VNode，无需处理字符串
- ✅ 代码更简洁
- ✅ 符合 Vue 3 最佳实践

### 3. SearchInput.vue - inputConfig

#### v1 的实现方式
```typescript
// 创建临时 Widget，调用方法，返回配置对象
const tempWidget = WidgetBuilder.createTemporary({
  field: props.field
})

return tempWidget.renderSearchInput(props.searchType)
// 返回：{ component: 'ElInput', props: {...}, onRemoteMethod: ... }
```

#### v2 的重构思路

**问题分析**：
- v1 返回配置对象，需要动态渲染
- v2 是 Vue 组件，可以直接使用

**重构方案A（推荐）：直接使用组件**
```vue
<template>
  <component 
    v-if="widgetComponent"
    :is="widgetComponent"
    :field="field"
    :value="fieldValue"
    :model-value="fieldValue"
    :field-path="field.code"
    mode="search"
    :search-type="searchType"
    @update:model-value="handleUpdate"
  />
</template>

<script setup>
import { computed } from 'vue'
import { widgetComponentFactory } from '@/core/factories-v2'

const widgetComponent = computed(() => {
  return widgetComponentFactory.getRequestComponent(
    props.field.widget?.type || 'input'
  )
})

const fieldValue = computed(() => ({
  raw: props.modelValue,
  display: props.modelValue,
  meta: {}
}))

const handleUpdate = (value: FieldValue) => {
  emit('update:modelValue', value.raw)
}
</script>
```

**重构方案B（兼容现有逻辑）：适配层**
如果 SearchInput 的逻辑比较复杂，可以创建适配层，但最终目标还是直接使用组件。

**改进点**：
- ✅ 直接使用组件，无需配置对象
- ✅ 更符合 Vue 3 组件化思想
- ✅ 代码更简洁

## 🎨 v2 的设计理念

### 1. 统一接口

**v1**：多个方法对应不同场景
```typescript
widget.renderTableCell()
widget.renderForDetail()
widget.renderSearchInput()
```

**v2**：统一的 Props 接口，通过 `mode` 区分
```typescript
<WidgetComponent mode="table-cell" />
<WidgetComponent mode="detail" />
<WidgetComponent mode="search" />
```

### 2. 组件化

**v1**：基于类，需要创建实例
```typescript
const widget = WidgetBuilder.createTemporary({ field, value })
const result = widget.renderTableCell()
```

**v2**：基于 Vue 组件，直接使用
```vue
<component :is="widgetComponent" :field="field" :value="value" />
```

### 3. 状态管理

**v1**：通过 formManager 传递状态
```typescript
const widget = WidgetBuilder.create({
  field: field,
  formManager: formManager
})
```

**v2**：使用 Pinia Store
```typescript
const formDataStore = useFormDataStore()
// 直接访问 store，无需传递
```

### 4. 组合式函数

**v1**：逻辑封装在类中
```typescript
class UserWidget extends BaseWidget {
  renderTableCell() {
    // 逻辑在类中
  }
}
```

**v2**：使用 composables 提取共享逻辑
```typescript
// composables/useUserWidget.ts
export function useUserWidget(props) {
  // 共享逻辑
}

// UserWidget.vue
<script setup>
const { userInfo, displayName } = useUserWidget(props)
</script>
```

## 🔧 重构实施步骤

### Step 1: 理解 v2 的设计理念（1小时）

1. **阅读 v2 的代码**
   - 查看 `widgets-v2/components/UserWidget.vue` 的实现
   - 理解 `mode` prop 的使用方式
   - 理解 composables 的使用方式

2. **对比 v1 和 v2**
   - 找出 v1 的问题
   - 理解 v2 的优势
   - 确定重构方向

### Step 2: 重构 TableRenderer（2-3小时）

1. **重构 renderTableCell**
   - 按照 v2 的思路重新实现
   - 使用 `widgetComponentFactory` + `h()` 渲染组件
   - 统一返回 VNode

2. **重构 renderDetailField**
   - 按照 v2 的思路重新实现
   - 使用 `mode="detail"`

3. **测试验证**
   - 测试所有 Widget 类型
   - 测试边界情况

### Step 3: 重构 SearchInput（1-2小时）

1. **重构 inputConfig**
   - 方案A：直接使用组件（推荐）
   - 方案B：创建适配层（如果需要）

2. **测试验证**
   - 测试所有搜索类型
   - 测试所有 Widget 类型

### Step 4: 清理和优化（1小时）

1. **删除旧代码**
   - 删除 `widgets/` 目录
   - 删除旧工厂

2. **代码优化**
   - 检查是否有可以优化的地方
   - 统一代码风格

## ⚠️ 注意事项

1. **不是复制代码**：要按照 v2 的思路重新实现
2. **理解设计理念**：先理解 v2 的设计，再重构
3. **保证可用性**：重构后要保证功能正常
4. **提高维护性**：重构的目标是提高代码质量

## 📚 参考

- `web/src/core/widgets-v2/components/UserWidget.vue` - v2 的实现示例
- `web/src/core/renderers-v2/FormRenderer.vue` - v2 的使用示例
- `web/docs/新旧版本Widget系统深度对比分析.md` - 对比分析

