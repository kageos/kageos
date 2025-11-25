# 迁移 TableRenderer 和 SearchInput 到 widgets-v2 方案

## 🎯 发现

**重要发现**：widgets-v2 组件已经支持 `table-cell` 和 `detail` 模式！

- ✅ `WidgetMode = 'edit' | 'response' | 'table-cell' | 'detail' | 'search'`
- ✅ 所有 widgets-v2 组件都实现了这些模式
- ✅ 可以直接用于 TableRenderer 和 SearchInput

## 📊 当前状态

### ✅ 已使用 widgets-v2
- **FormRenderer（renderers-v2）** - 完全使用 widgets-v2

### ⚠️ 仍使用旧版本
- **TableRenderer** - 使用 `WidgetBuilder.createTemporary()` + `renderTableCell()`
- **SearchInput** - 使用 `WidgetBuilder.createTemporary()` + `renderSearchInput()`

## 🔄 迁移方案

### 方案1：直接替换（推荐）

#### TableRenderer 迁移

**当前代码**：
```typescript
// 旧版本
const tempWidget = WidgetBuilder.createTemporary({
  field: coreField,
  value: value
})
const result = tempWidget.renderTableCell(value, userInfoMap.value)
```

**迁移后**：
```vue
<template>
  <el-table-column>
    <template #default="{ row }">
      <component 
        :is="getWidgetComponent(field.widget?.type || 'input')"
        :field="field"
        :value="convertToFieldValue(row[field.code], field)"
        :model-value="convertToFieldValue(row[field.code], field)"
        :field-path="field.code"
        mode="table-cell"
        :user-info-map="userInfoMap"
      />
    </template>
  </el-table-column>
</template>

<script setup>
import { widgetComponentFactory } from '@/core/factories-v2'

function getWidgetComponent(type: string) {
  return widgetComponentFactory.getRequestComponent(type) || 
         widgetComponentFactory.getRequestComponent('input')
}
</script>
```

#### SearchInput 迁移

**当前代码**：
```typescript
// 旧版本
const tempWidget = WidgetBuilder.createTemporary({
  field: props.field
})
return tempWidget.renderSearchInput(props.searchType)
```

**迁移后**：
```vue
<template>
  <component 
    v-if="widgetComponent"
    :is="widgetComponent"
    :field="field"
    :value="getFieldValue()"
    :model-value="getFieldValue()"
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
  return widgetComponentFactory.getRequestComponent(props.field.widget?.type || 'input')
})
</script>
```

### 方案2：创建适配函数（兼容性更好）

如果直接替换有困难，可以创建适配函数：

```typescript
// utils/widgetAdapter.ts
import { h } from 'vue'
import { widgetComponentFactory } from '@/core/factories-v2'
import { convertToFieldValue } from '@/utils/field'

/**
 * 使用 widgets-v2 渲染表格单元格
 */
export function renderTableCellWithV2(
  field: FieldConfig,
  rawValue: any,
  userInfoMap?: Map<string, any>
): any {
  const value = convertToFieldValue(rawValue, field)
  const WidgetComponent = widgetComponentFactory.getRequestComponent(
    field.widget?.type || 'input'
  )
  
  if (!WidgetComponent) {
    return rawValue !== null && rawValue !== undefined ? String(rawValue) : '-'
  }
  
  return h(WidgetComponent, {
    field,
    value,
    'model-value': value,
    'field-path': field.code,
    mode: 'table-cell',
    'user-info-map': userInfoMap
  })
}

/**
 * 使用 widgets-v2 获取搜索输入配置
 */
export function getSearchInputConfigWithV2(
  field: FieldConfig,
  searchType: string
): any {
  const WidgetComponent = widgetComponentFactory.getRequestComponent(
    field.widget?.type || 'input'
  )
  
  if (!WidgetComponent) {
    return {
      component: 'ElInput',
      props: {
        placeholder: `请输入${field.name}`,
        clearable: true
      }
    }
  }
  
  // 需要从组件中提取配置
  // 可以通过 props 或 composable 获取
  // 这里需要根据实际情况实现
}
```

## 🎯 迁移步骤

### Step 1: 更新 TableRenderer

1. 导入 `widgetComponentFactory`
2. 替换 `renderTableCell` 函数
3. 替换 `renderDetailField` 函数
4. 测试表格渲染

### Step 2: 更新 SearchInput

1. 导入 `widgetComponentFactory`
2. 替换 `inputConfig` computed
3. 测试搜索输入

### Step 3: 清理旧代码

1. 删除 `WidgetBuilder` 的导入
2. 删除旧版本的调用
3. 测试所有功能

## ⚠️ 注意事项

1. **VNode vs 字符串**：
   - 旧版本返回 VNode 或字符串
   - widgets-v2 组件返回 VNode
   - 需要确保表格能正确渲染 VNode

2. **搜索配置**：
   - 旧版本返回配置对象
   - widgets-v2 是组件，需要提取配置或直接使用组件

3. **用户信息映射**：
   - widgets-v2 支持 `user-info-map` prop
   - 可以直接传递 `userInfoMap`

## ✅ 优势

1. **统一版本**：所有场景都使用 widgets-v2
2. **减少维护**：只需要维护一套系统
3. **功能一致**：表格和表单使用相同的组件
4. **易于扩展**：新功能只需要在 widgets-v2 中添加

## 📝 总结

**结论**：TableRenderer 和 SearchInput **可以**迁移到 widgets-v2！

- ✅ widgets-v2 已经支持 `table-cell` 和 `detail` 模式
- ✅ 可以直接替换，或者创建适配函数
- ✅ 迁移后可以删除旧版本 widgets/ 目录

**建议**：开始迁移，统一使用 widgets-v2！

