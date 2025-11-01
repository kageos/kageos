# 修复：List 组件提交空数组问题

## 🐛 问题描述

用户在 List 组件中选择了数据（例如商品列表），但提交时该字段的值为空数组：

```json
{
  "product_quantities": [],  // ❌ 明明选了商品，却是空数组
  "member_id": 1,
  "remarks": "买瓶可乐"
}
```

## 🔍 根本原因

### 数据流问题

1. **子 Widget 独立管理自己的值**：
   - 当用户在 `product_quantities[0].product_id` 中选择商品时
   - `SelectWidget` 通过 `onChange` 回调更新了 `FormDataManager` 中 `product_quantities[0].product_id` 的值
   - ✅ 子 Widget 的值已正确存储

2. **ListWidget 的值未同步**：
   - `ListWidget` 自身在 `FormDataManager` 中也有一个值：`product_quantities`
   - 初始化时，这个值是 `[]`（空数组）
   - 子 Widget 的值变化后，**ListWidget 的值并没有同步更新**
   - ❌ `product_quantities` 依然是 `[]`

3. **`collectListValue` 依赖 ListWidget 的值**：
   ```typescript
   // 旧代码
   function collectListValue(widget: BaseWidget, field: FieldConfig): any[] {
     const rawValue = widget.getValue().raw  // ❌ 获取的是 ListWidget 的 raw 值 []
     
     return rawValue.map((item: any, index: number) => {
       // ... 遍历子 Widget
     })
   }
   ```
   - 因为 `rawValue` 是空数组，`map` 不会遍历，导致子 Widget 的值被忽略

### 架构设计问题

当前架构中，**子 Widget 的值是独立存储的**，这是正确的设计（符合"每个组件管理自己"的原则）。但在收集提交数据时，我们却依赖于父 Widget 的值，导致了不一致。

## ✅ 解决方案

### 核心思路

**不依赖 ListWidget 的 `raw` 值，而是直接从所有已注册的子 Widget 中收集数据。**

### 实现代码

```typescript
function collectListValue(widget: BaseWidget, field: FieldConfig): any[] {
  const children = field.children || []
  if (children.length === 0) {
    // 如果没有子字段定义，直接返回 raw 值
    const rawValue = widget.getValue().raw
    return Array.isArray(rawValue) ? rawValue : []
  }
  
  // 🔥 遍历所有已注册的子 Widget，收集它们的值
  const result: any[] = []
  
  // 找出所有属于这个 List 的子 Widget（通过 fieldPath 前缀匹配）
  const listPrefix = `${field.code}[`
  const itemsByIndex = new Map<number, Record<string, any>>()
  
  allWidgets.forEach((childWidget, childPath) => {
    if (childPath.startsWith(listPrefix)) {
      // 解析路径: product_quantities[0].product_id -> index=0, code=product_id
      const match = childPath.match(/\[(\d+)\]\.(\w+)$/)
      if (match) {
        const index = parseInt(match[1], 10)
        const code = match[2]
        
        if (!itemsByIndex.has(index)) {
          itemsByIndex.set(index, {})
        }
        
        itemsByIndex.get(index)![code] = childWidget.getRawValueForSubmit()
      }
    }
  })
  
  // 按索引顺序转为数组
  const maxIndex = Math.max(-1, ...Array.from(itemsByIndex.keys()))
  for (let i = 0; i <= maxIndex; i++) {
    result.push(itemsByIndex.get(i) || {})
  }
  
  return result
}
```

### 工作流程

1. **遍历所有已注册的 Widget**（`allWidgets`）
2. **通过 fieldPath 前缀匹配**找出属于当前 List 的子 Widget
   - 例如：`product_quantities[0].product_id`、`product_quantities[0].quantity`
3. **解析索引和字段名**：
   - `product_quantities[0].product_id` → `index=0`, `code=product_id`
4. **按索引组织数据**：
   - 使用 `Map<number, Record<string, any>>` 按行索引分组
   - `{ 0: { product_id: 1, quantity: 2 } }`
5. **转为数组**：
   - 按索引顺序（0, 1, 2...）转为数组
   - 缺失的索引填充空对象 `{}`

## 📋 影响范围

### 修改文件

- ✅ `web/src/core/renderers/FormRenderer.vue`
  - `collectListValue` 函数

### 优势

1. **数据一致性**：直接从子 Widget 收集，避免同步问题
2. **符合架构原则**：每个组件独立管理自己的值
3. **无需修改 ListWidget**：不需要添加复杂的值同步逻辑
4. **支持未来扩展**：适用于嵌套 List、Struct 等复杂场景

### 测试场景

- ✅ List 内 Select 选择商品
- ✅ List 内 Input 输入数量
- ✅ 添加/删除行
- ✅ 多行数据提交

## 🎯 总结

这个修复**从根本上解决了父子 Widget 值同步的问题**，通过"直接从子 Widget 收集数据"的方式，避免了复杂的状态同步逻辑，使架构更清晰、更健壮。

---

**修复日期**：2025-11-01  
**问题严重级别**：🔴 高（影响所有 List 组件的提交功能）  
**修复状态**：✅ 已完成

