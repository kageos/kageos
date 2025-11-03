# TableRenderer 重构方案分析

## 🎯 核心需求

1. **字段详情展示 = Form 渲染**：详情抽屉里的字段应该使用组件的 Form 渲染能力
2. **组件自治的 Table 展示**：每个组件可以重写 `renderTableCell()` 方法自定义表格展示
3. **依赖倒置原则**：TableRenderer 依赖抽象（Widget 接口），不依赖具体实现
4. **扩展性**：新增组件时，只需：
   - 实现 `renderTableCell()` → 自定义表格展示
   - 实现 `render()` → 自定义表单展示
   - **无需修改 TableRenderer**

## 📊 当前架构分析

### ✅ 已有机制（符合依赖倒置）

```
BaseWidget.renderTableCell(value: FieldValue)
    ↓
子组件可以重写
    ↓
MultiSelectWidget.renderTableCell() ✅ 已实现
FileWidget.renderTableCell()        ⬜ 待实现
    ↓
TableWidget.renderCellByWidget()   ✅ 已使用
```

### ❌ TableRenderer 的问题

#### 1. 表格单元格硬编码
```typescript
// ❌ 当前：硬编码逻辑
<template #default="{ row, $index }">
  <span v-if="isIdColumn(field)">...</span>
  <span v-else-if="field.widget.type === 'timestamp'">
    {{ formatTimestamp(row[field.code], ...) }}  // 硬编码
  </span>
  <span v-else>{{ row[field.code] }}</span>      // 硬编码
</template>
```

**问题**：
- 新增文件组件时，需要修改 TableRenderer
- 时间戳格式化逻辑重复（与 BaseWidget 重复）
- 违反依赖倒置原则

#### 2. 详情抽屉硬编码
```typescript
// ❌ 当前：硬编码展示
<el-descriptions-item>
  <template v-if="field.widget.type === 'timestamp'">
    {{ formatTimestamp(...) }}  // 硬编码
  </template>
  <template v-else>
    {{ currentDetailRow[field.code] || '-' }}  // 硬编码
  </template>
</el-descriptions-item>
```

**问题**：
- 文件组件在详情里无法展示（应该显示文件列表/预览）
- Select 组件应该显示 label 而不是 raw 值
- 与 Form 渲染不一致

## 🏗️ 重构方案

### 方案架构图

```
TableRenderer.vue (UI 层)
    ↓ 依赖注入
useTableOperations (业务逻辑层)
    ↓ 依赖抽象
WidgetFactory → Widget.renderTableCell() (组件层)
```

### 1️⃣ 表格单元格渲染（组件自治）

```typescript
// ✅ TableRenderer.vue
<template #default="{ row, $index }">
  <!-- ID 列特殊处理 -->
  <span v-if="isIdColumn(field)" @click="...">
    {{ row[field.code] }}
  </span>
  <!-- 🔥 其他列：使用 Widget 的 renderTableCell -->
  <component 
    v-else
    :is="renderTableCell(field, row[field.code])"
  />
</template>

// ✅ 使用 Widget 渲染
const renderTableCell = (field: FieldConfig, rawValue: any) => {
  const value = convertToFieldValue(rawValue, field)
  const tempWidget = WidgetBuilder.createTemporary({ field, value })
  return tempWidget.renderTableCell(value)  // 🔥 组件自治
}
```

**优势**：
- ✅ FileWidget 只需实现 `renderTableCell()` 就能自定义展示
- ✅ 新增组件无需修改 TableRenderer
- ✅ 符合依赖倒置原则

### 2️⃣ 详情抽屉渲染（复用 Form 渲染引擎）

```typescript
// ✅ 详情抽屉使用 Widget 的 render() 方法
<div class="detail-content">
  <el-descriptions :column="1" border>
    <el-descriptions-item
      v-for="field in visibleFields"
      :key="field.code"
      :label="field.name"
    >
      <!-- 🔥 使用 Widget 的 render() 方法（只读模式） -->
      <component 
        :is="renderDetailField(field, currentDetailRow[field.code])"
      />
    </el-descriptions-item>
  </el-descriptions>
</div>

// ✅ 使用 Widget 渲染（与 Form 一致）
const renderDetailField = (field: FieldConfig, rawValue: any) => {
  const value = convertToFieldValue(rawValue, field)
  const tempWidget = WidgetBuilder.createTemporary({ 
    field, 
    value,
    readonly: true  // 只读模式
  })
  return tempWidget.render()  // 🔥 复用 Form 渲染引擎
}
```

**优势**：
- ✅ 详情展示与 Form 完全一致
- ✅ FileWidget 在详情中自动显示文件列表/预览
- ✅ SelectWidget 显示 label 而不是 raw 值
- ✅ 无需重复实现详情渲染逻辑

### 3️⃣ 业务逻辑抽离（useTableOperations）

```typescript
// ✅ composables/useTableOperations.ts
export function useTableOperations(functionData: FunctionType) {
  // 状态
  const loading = ref(false)
  const tableData = ref<any[]>([])
  const searchForm = ref<Record<string, any>>({})
  const pagination = ref({ page: 1, pageSize: 20, total: 0 })
  
  // 业务逻辑
  const loadData = async () => { ... }
  const search = () => { ... }
  const reset = () => { ... }
  const add = async (data: any) => { ... }
  const update = async (id: number, data: any) => { ... }
  const deleteRow = async (id: number) => { ... }
  
  return {
    // 状态
    loading,
    tableData,
    searchForm,
    pagination,
    // 方法
    loadData,
    search,
    reset,
    add,
    update,
    deleteRow
  }
}
```

**优势**：
- ✅ 业务逻辑可复用
- ✅ 易于测试
- ✅ TableRenderer 只负责 UI

### 4️⃣ 工具函数抽离

```typescript
// ✅ utils/date.ts
export function formatTimestamp(timestamp: number, format?: string): string {
  // 统一的时间戳格式化
}

// ✅ utils/field.ts
export function convertToFieldValue(rawValue: any, field: FieldConfig): FieldValue {
  // 统一的值转换
}
```

## 📋 重构步骤

### Phase 1: 基础架构（依赖倒置）
1. ✅ 创建 `useTableOperations` composable
2. ✅ 抽离工具函数（formatTimestamp, convertToFieldValue）
3. ✅ 表格单元格使用 `Widget.renderTableCell()`
4. ✅ 详情抽屉使用 `Widget.render()`（只读模式）

### Phase 2: 业务逻辑解耦
5. ✅ 将搜索、分页、排序逻辑移到 composable
6. ✅ 将 CRUD 操作逻辑移到 composable
7. ✅ TableRenderer 只负责 UI 渲染

### Phase 3: 扩展性验证
8. ✅ 创建 FileWidget 示例
9. ✅ 实现 `FileWidget.renderTableCell()` → 显示文件图标/数量
10. ✅ 验证无需修改 TableRenderer

## 🎯 最终效果

### 新增文件组件时：
```typescript
// ✅ 只需新增 FileWidget.ts
class FileWidget extends BaseWidget {
  // 表格展示：显示文件数量和图标
  renderTableCell(value: FieldValue): any {
    const files = value.raw || []
    return h('div', [
      h(ElIcon, { File }),
      h('span', `共 ${files.length} 个文件`)
    ])
  }
  
  // 表单展示：文件上传组件
  render() {
    return h(ElUpload, { ... })
  }
}
```

**无需修改 TableRenderer！** ✅

## ✅ 架构优势

| 维度 | 当前 | 重构后 |
|------|------|--------|
| **依赖倒置** | ❌ 依赖具体实现 | ✅ 依赖抽象（Widget 接口） |
| **扩展性** | ❌ 新增组件需改 TableRenderer | ✅ 只需实现 Widget 方法 |
| **维护性** | ❌ 逻辑分散 | ✅ 逻辑集中在 composable |
| **一致性** | ❌ Table/Form/Detail 展示不一致 | ✅ 统一使用 Widget 渲染 |
| **可测试性** | ❌ 难以测试 | ✅ composable 易于测试 |

## 🚀 下一步

开始重构？

