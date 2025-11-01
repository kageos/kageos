# 重构：FormDialog 复用 FormRenderer

## 🎯 重构目标

消除 `FormDialog.vue` 和 `FormRenderer.vue` 之间的重复代码，让 `FormDialog` 复用 `FormRenderer` 的渲染引擎。

---

## 💡 核心洞察

**Table 的新增/编辑表单** 和 **Form 函数** 的字段结构**完全一致**：
- 都是 `FieldConfig[]`
- 都需要处理默认值、验证、回调
- 都需要渲染相同的 Widget（Input、Select、TextArea、List 等）

---

## 🔴 重构前

### FormDialog.vue（旧版）
- **474 行代码**
- 大量重复的渲染逻辑：

```vue
<!-- ❌ 重复的 Widget 渲染逻辑 -->
<el-input v-if="field.widget.type === 'input'" ... />
<el-input-number v-else-if="field.widget.type === 'number'" ... />
<el-input v-else-if="field.widget.type === 'text_area'" type="textarea" ... />
<el-select v-else-if="field.widget.type === 'select'" ... />
<el-date-picker v-else-if="field.widget.type === 'timestamp'" ... />
<el-switch v-else-if="field.widget.type === 'switch'" ... />
<el-checkbox-group v-else-if="field.widget.type === 'checkbox'" ... />
<el-radio-group v-else-if="field.widget.type === 'radio'" ... />
<!-- ... 更多 -->
```

- 重复的默认值初始化逻辑
- 重复的验证规则解析逻辑
- **不支持**：嵌套结构（List、Struct）、回调、聚合统计

---

## 🟢 重构后

### FormDialog.vue（新版）
- **约 160 行代码**（减少 66%）
- 直接复用 `FormRenderer`：

```vue
<template>
  <el-dialog v-model="dialogVisible" :title="title" :width="width">
    <!-- ✅ 复用 FormRenderer -->
    <FormRenderer
      v-if="dialogVisible"
      ref="formRendererRef"
      :function-detail="formFunctionDetail"
      :show-submit-button="false"
      :show-share-button="false"
      :show-reset-button="false"
      :show-debug-button="false"
    />

    <template #footer>
      <el-button @click="handleClose">取消</el-button>
      <el-button type="primary" @click="handleSubmit" :loading="submitting">
        确定
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
// 🔥 将 fields 包装成 FunctionDetail 格式
const formFunctionDetail = computed<FunctionDetail>(() => ({
  id: 0,
  method: 'POST',
  router: '',
  template_type: 'form',
  request: filteredFields.value,  // 使用过滤后的字段
  response: []
}))

// 🔥 调用 FormRenderer 的方法提交
const handleSubmit = async () => {
  const submitData = formRendererRef.value.prepareSubmitDataWithTypeConversion()
  emit('submit', submitData)
}
</script>
```

---

## 🔧 关键修改

### 1. **FormRenderer 添加控制 Props**

```typescript
// FormRenderer.vue
const props = withDefaults(defineProps<{
  functionDetail: FunctionDetail
  showSubmitButton?: boolean    // 控制提交按钮显示
  showShareButton?: boolean     // 控制分享按钮显示
  showResetButton?: boolean     // 控制重置按钮显示
  showDebugButton?: boolean     // 控制调试按钮显示
}>(), {
  showSubmitButton: true,
  showShareButton: true,
  showResetButton: true,
  showDebugButton: true
})
```

### 2. **FormRenderer 暴露方法**

```typescript
// FormRenderer.vue
defineExpose({
  prepareSubmitDataWithTypeConversion,  // 准备提交数据（带类型转换）
  formManager,                          // 表单数据管理器
  allWidgets,                           // 所有 Widget 实例
  handleRealSubmit                      // 真实提交方法
})
```

### 3. **FormDialog 包装 fields**

```typescript
// FormDialog.vue
const formFunctionDetail = computed<FunctionDetail>(() => ({
  id: 0,
  app_id: 0,
  tree_id: 0,
  method: 'POST',
  router: '',
  template_type: 'form',
  request: filteredFields.value,  // 🔥 使用过滤后的字段
  response: []
}))
```

### 4. **保留 table_permission 过滤逻辑**

```typescript
// FormDialog.vue
const filteredFields = computed(() => {
  return props.fields.filter(field => {
    const permission = field.table_permission
    
    if (props.mode === 'create') {
      // read: 不显示（后端自动生成）
      // update: 不显示（只能编辑时修改）
      // create: 显示（只能新增时填写）
      // 空: 显示（全部权限）
      return !permission || permission === '' || permission === 'create'
    }
    
    if (props.mode === 'update') {
      // read: 不显示（只读）
      // update: 显示（只能编辑时修改）
      // create: 不显示（只能新增时填写）
      // 空: 显示（全部权限）
      return !permission || permission === '' || permission === 'update'
    }
    
    return true
  })
})
```

---

## ✅ 重构优势

### 1. **大幅减少代码量**
- ❌ 删除 474 行 → ✅ 仅需 160 行
- 减少 **66%** 的代码

### 2. **保持行为一致性**
- Form 函数和 Table 新增/编辑使用**完全相同**的渲染引擎
- 新增 Widget 类型时，**两边自动生效**
- Bug 修复和功能增强**一次完成**

### 3. **自动支持高级功能**
- ✅ **嵌套结构**：List、Struct（旧版不支持）
- ✅ **回调系统**：OnSelectFuzzy（旧版不支持）
- ✅ **聚合统计**：List 内 Select/MultiSelect 聚合（旧版不支持）
- ✅ **快照/分享**：表单状态持久化（旧版不支持）
- ✅ **类型转换**：自动转换 string → int/float/bool（旧版需要手动处理）

### 4. **易于维护**
- 只需维护一个 `FormRenderer`
- Widget 逻辑集中在 `BaseWidget` 及其子类
- 符合 **单一职责** 和 **开闭原则**

---

## 📊 对比总结

| 维度 | 重构前 | 重构后 |
|------|--------|--------|
| **代码行数** | 474 行 | 160 行（-66%） |
| **渲染逻辑** | 重复实现 | 复用 FormRenderer |
| **嵌套结构** | ❌ 不支持 | ✅ 支持 |
| **回调系统** | ❌ 不支持 | ✅ 支持 |
| **聚合统计** | ❌ 不支持 | ✅ 支持 |
| **快照/分享** | ❌ 不支持 | ✅ 支持 |
| **类型转换** | ❌ 手动处理 | ✅ 自动处理 |
| **新增 Widget** | 需修改 FormDialog | 自动生效 |
| **维护成本** | 高（两处修改） | 低（一处修改） |

---

## 🧪 测试场景

### 1. **Table 新增记录**
```typescript
// TableRenderer.vue
<FormDialog
  v-model="addDialogVisible"
  title="新增记录"
  :fields="props.functionDetail.request"
  mode="create"
  @submit="handleAddSubmit"
/>
```

### 2. **Table 编辑记录**
```typescript
// TableRenderer.vue
<FormDialog
  v-model="editDialogVisible"
  title="编辑记录"
  :fields="props.functionDetail.request"
  mode="update"
  :initial-data="currentRow"
  @submit="handleEditSubmit"
/>
```

### 3. **嵌套结构（List 内 Select）**
```typescript
// 自动支持！无需修改代码
{
  code: "product_quantities",
  name: "商品清单",
  widget: { type: "table" },
  children: [
    {
      code: "product_id",
      name: "商品",
      widget: { type: "select" },
      callbacks: ["OnSelectFuzzy"]  // ✅ 自动支持
    },
    {
      code: "quantity",
      name: "数量",
      widget: { type: "number" }
    }
  ]
}
```

---

## 🚀 未来扩展

### 1. **添加新 Widget 类型**
```typescript
// 只需创建新的 Widget 类
export class DateRangeWidget extends BaseWidget {
  static getDefaultValue(field: FieldConfig): FieldValue {
    // ... 默认值逻辑
  }
  
  render() {
    // ... 渲染逻辑
  }
}

// ✅ FormDialog 和 FormRenderer 自动支持！
```

### 2. **支持更多回调**
```typescript
// 只需在 Widget 中添加回调处理
export class InputWidget extends BaseWidget {
  async handleValidate() {
    // 调用 OnInputValidate 回调
  }
}

// ✅ FormDialog 和 FormRenderer 自动支持！
```

### 3. **支持条件显示**
```typescript
// ConditionEvaluator 解析 validation
const shouldShow = ConditionEvaluator.evaluate(
  'required_if=member_id,!=""',
  formManager.prepareSubmitData()
)

// ✅ FormDialog 和 FormRenderer 自动支持！
```

---

## 📝 总结

本次重构通过**组件复用**的设计模式，消除了 `FormDialog` 和 `FormRenderer` 之间的重复代码。

核心思想：
1. **识别重复**：Table 新增/编辑和 Form 函数结构一致
2. **提取公共**：FormRenderer 作为通用渲染引擎
3. **包装适配**：FormDialog 将 fields 包装成 FunctionDetail
4. **保留特性**：table_permission 过滤逻辑在 FormDialog 中保留

结果：
- ✅ 代码减少 66%
- ✅ 自动支持所有高级功能
- ✅ 易于维护和扩展
- ✅ 符合 OOP 设计原则

这为后续功能开发（回调、聚合、验证、条件显示）提供了坚实的架构基础。🎉

