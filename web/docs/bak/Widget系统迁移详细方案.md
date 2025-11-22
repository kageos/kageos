# Widget 系统重构详细方案

## ⚠️ 重要说明

**这不是迁移，而是重构**：按照 widgets-v2 的设计思路和风格重新实现，而不是简单复制代码。

### 核心原则
1. **按照 v2 的思路重新实现**：不是复制代码，而是理解 v2 的设计理念后重新实现
2. **保证可用性和维护性**：重构的目标是提高代码质量和可维护性
3. **v1 不一定是最合适的**：v1 的实现可能有历史包袱，需要重新审视和优化

## 📋 迁移点完整清单

### 核心迁移点（必须迁移）

#### 1. TableRenderer.vue - renderTableCell

**文件**：`web/src/components/TableRenderer.vue`  
**位置**：第 524-564 行  
**当前代码**：
```typescript
const renderTableCell = (field: FieldConfig, rawValue: any): { content: any, isString: boolean } => {
  const tempWidget = WidgetBuilder.createTemporary({
    field: coreField,
    value: value
  })
  const result = tempWidget.renderTableCell(value, userInfoMap.value)
  // ...
}
```

**迁移后**：
```typescript
import { widgetComponentFactory } from '@/core/factories-v2'
import { h } from 'vue'

const renderTableCell = (field: FieldConfig, rawValue: any): { content: any, isString: boolean } => {
  try {
    const value = convertToFieldValue(rawValue, field)
    const WidgetComponent = widgetComponentFactory.getRequestComponent(field.widget?.type || 'input')
    
    if (!WidgetComponent) {
      return {
        content: rawValue !== null && rawValue !== undefined ? String(rawValue) : '-',
        isString: true
      }
    }
    
    // 使用 h() 渲染组件为 VNode
    const vnode = h(WidgetComponent, {
      field: field,
      value: value,
      'model-value': value,
      'field-path': field.code,
      mode: 'table-cell',
      'user-info-map': userInfoMap.value
    })
    
    return {
      content: vnode,
      isString: false,
      isVNode: true
    }
  } catch (error) {
    // 错误处理
  }
}
```

#### 2. TableRenderer.vue - renderDetailField

**文件**：`web/src/components/TableRenderer.vue`  
**位置**：第 590-625 行  
**当前代码**：
```typescript
const renderDetailField = (field: FieldConfig, rawValue: any): any => {
  const widget = WidgetBuilder.createTemporary({
    field: field,
    value: value
  })
  const result = widget.renderForDetail(value, context)
  // ...
}
```

**迁移后**：
```typescript
const renderDetailField = (field: FieldConfig, rawValue: any): any => {
  try {
    const value = convertToFieldValue(rawValue, field)
    const WidgetComponent = widgetComponentFactory.getRequestComponent(field.widget?.type || 'input')
    
    if (!WidgetComponent) {
      return h('span', rawValue !== null && rawValue !== undefined ? String(rawValue) : '-')
    }
    
    const context = {
      functionName: props.currentFunction?.name || props.currentFunction?.code || '',
      recordId: currentDetailRow.value?.id || currentDetailRow.value?.[idField.value?.code || 'id'],
      userInfoMap: userInfoMap.value
    }
    
    return h(WidgetComponent, {
      field: field,
      value: value,
      'model-value': value,
      'field-path': field.code,
      mode: 'detail',
      'user-info-map': userInfoMap.value
    })
  } catch (error) {
    // 错误处理
  }
}
```

#### 3. SearchInput.vue - inputConfig

**文件**：`web/src/components/SearchInput.vue`  
**位置**：第 332-355 行  
**当前代码**：
```typescript
const inputConfig = computed(() => {
  const tempWidget = WidgetBuilder.createTemporary({
    field: props.field
  })
  return (tempWidget as any).renderSearchInput(props.searchType)
})
```

**迁移方案A（推荐）：直接使用组件**
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

const getFieldValue = () => {
  // 根据当前值类型返回 FieldValue
  return {
    raw: props.modelValue,
    display: props.modelValue,
    meta: {}
  }
}

const handleUpdate = (value: FieldValue) => {
  emit('update:modelValue', value.raw)
}
</script>
```

**迁移方案B（兼容现有逻辑）：适配层**
```typescript
import { widgetComponentFactory } from '@/core/factories-v2'
import { h } from 'vue'

const inputConfig = computed(() => {
  try {
    const WidgetComponent = widgetComponentFactory.getRequestComponent(props.field.widget?.type || 'input')
    
    if (!WidgetComponent) {
      return {
        component: 'ElInput',
        props: {
          placeholder: `请输入${props.field.name}`,
          clearable: true,
          style: { width: '200px' }
        }
      }
    }
    
    // 对于 search 模式，需要特殊处理
    // 如果组件支持 search 模式，直接使用
    // 否则返回配置对象（兼容现有逻辑）
    
    // 检查组件是否支持 search 模式
    // 如果支持，返回组件配置
    return {
      component: WidgetComponent,
      props: {
        field: props.field,
        mode: 'search',
        searchType: props.searchType
      }
    }
  } catch (error) {
    // 错误处理
  }
})
```

### 内部使用点（需要检查）

#### 4. ResponseTableWidget.ts

**文件**：`web/src/core/widgets/ResponseTableWidget.ts`  
**状态**：需要检查是否还在使用

**检查方法**：
```bash
grep -r "ResponseTableWidget" web/src --exclude-dir=node_modules
```

**如果使用**：
- 需要迁移到 widgets-v2
- 或者使用新的 FormRenderer（renderers-v2）

**如果不使用**：
- 可以删除

#### 5. ResponseFormWidget.ts

**文件**：`web/src/core/widgets/ResponseFormWidget.ts`  
**状态**：需要检查是否还在使用

**检查方法**：
```bash
grep -r "ResponseFormWidget" web/src --exclude-dir=node_modules
```

**如果使用**：
- 需要迁移到 widgets-v2
- 或者使用新的 FormRenderer（renderers-v2）

**如果不使用**：
- 可以删除

### 其他使用点

#### 6. field.ts

**文件**：`web/src/utils/field.ts`  
**使用**：`widgetFactory`  
**检查**：是否还在使用

#### 7. TableWidget.ts / FormWidget.ts

**文件**：`web/src/core/widgets/TableWidget.ts`、`FormWidget.ts`  
**状态**：这些是嵌套 Widget，内部使用旧版本  
**迁移方案**：如果还在使用，需要迁移到 widgets-v2 的 TableWidget/FormWidget

## 🔧 迁移实施步骤

### Step 1: 准备阶段（30分钟）

1. **创建迁移分支**
   ```bash
   git checkout -b migrate/widgets-v2
   git push -u origin migrate/widgets-v2
   ```

2. **检查所有使用点**
   ```bash
   # 检查 WidgetBuilder 使用
   grep -r "WidgetBuilder" web/src --exclude-dir=node_modules
   
   # 检查 WidgetFactory 使用
   grep -r "WidgetFactory" web/src --exclude-dir=node_modules
   
   # 检查旧版本 widgets 导入
   grep -r "from.*core/widgets/" web/src --exclude-dir=node_modules
   ```

3. **确认迁移范围**
   - 列出所有需要迁移的文件
   - 确认每个文件的迁移方案

### Step 2: 迁移 TableRenderer（2-3小时）

1. **更新导入**
   ```typescript
   // 删除
   import { WidgetBuilder } from '@/core/factories/WidgetBuilder'
   
   // 添加
   import { widgetComponentFactory } from '@/core/factories-v2'
   ```

2. **迁移 renderTableCell**
   - 按照上面的方案迁移
   - 测试表格渲染

3. **迁移 renderDetailField**
   - 按照上面的方案迁移
   - 测试详情展示

4. **测试验证**
   - 测试所有 Widget 类型
   - 测试边界情况

### Step 3: 迁移 SearchInput（1-2小时）

1. **更新导入**
   ```typescript
   // 删除
   import { WidgetBuilder } from '@/core/factories/WidgetBuilder'
   
   // 添加
   import { widgetComponentFactory } from '@/core/factories-v2'
   ```

2. **迁移 inputConfig**
   - 选择方案A（推荐）或方案B
   - 测试所有搜索类型

3. **测试验证**
   - 测试所有 Widget 类型
   - 测试所有搜索类型

### Step 4: 检查其他使用点（1小时）

1. **检查 ResponseTableWidget/ResponseFormWidget**
   - 如果使用，迁移或删除
   - 如果不使用，删除

2. **检查其他使用点**
   - field.ts
   - TableWidget.ts / FormWidget.ts

### Step 5: 清理和测试（1-2小时）

1. **删除旧代码**
   ```bash
   # 删除 widgets/ 目录
   rm -rf web/src/core/widgets/
   
   # 删除旧工厂
   rm web/src/core/factories/WidgetFactory.ts
   rm web/src/core/factories/WidgetBuilder.ts
   ```

2. **更新导入**
   - 检查所有文件，确保没有旧版本导入

3. **全面测试**
   - 功能测试
   - 性能测试
   - 边界测试

### Step 6: 文档和提交（30分钟）

1. **更新文档**
   - 更新 README
   - 更新架构文档

2. **提交代码**
   ```bash
   git add -A
   git commit -m "feat: 完全迁移到 widgets-v2，废弃 widgets/ v1"
   git push
   ```

## 🧪 测试清单

### 功能测试

#### TableRenderer
- [ ] 表格渲染正常
- [ ] 所有 Widget 类型单元格显示正常
  - [ ] input
  - [ ] number
  - [ ] text_area
  - [ ] select
  - [ ] multiselect
  - [ ] switch
  - [ ] timestamp
  - [ ] files
  - [ ] user
  - [ ] table（嵌套）
  - [ ] form（嵌套）
- [ ] 详情展示正常
- [ ] 用户信息显示正常（头像、昵称）
- [ ] 文件上传显示正常

#### SearchInput
- [ ] 所有搜索类型正常
  - [ ] eq（精确匹配）
  - [ ] like（模糊匹配）
  - [ ] in（多选）
  - [ ] gte/lte（范围）
- [ ] 所有 Widget 类型搜索输入正常
- [ ] 用户搜索输入正常（UserSearchInput）

### 边界测试

- [ ] 空值处理
- [ ] null/undefined 处理
- [ ] 错误处理
- [ ] 性能测试（大量数据）

## ⚠️ 注意事项

1. **一次迁移一个文件**：不要同时修改多个文件
2. **及时测试**：每个迁移点完成后立即测试
3. **保留旧代码**：迁移完成并验证后再删除
4. **文档更新**：及时更新相关文档
5. **代码审查**：迁移完成后进行代码审查

## 🔄 回滚方案

### 方案1：Git 回滚
```bash
# 如果迁移失败，回滚到迁移前
git reset --hard <迁移前commit>
git push --force
```

### 方案2：功能开关（如果需要）
```typescript
// 在配置中添加开关
const USE_WIDGETS_V2 = import.meta.env.VITE_USE_WIDGETS_V2 !== 'false'

// 在代码中使用
if (USE_WIDGETS_V2) {
  // 使用新版本
} else {
  // 使用旧版本（回滚）
}
```

## 📚 参考

- `web/docs/新旧版本Widget系统深度对比分析.md`
- `web/docs/迁移TableRenderer到widgets-v2方案.md`
- `web/src/core/widgets-v2/types.ts`
- `web/src/core/factories-v2/WidgetComponentFactory.ts`

