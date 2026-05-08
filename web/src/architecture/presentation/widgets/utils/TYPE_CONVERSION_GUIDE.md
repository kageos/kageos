# 类型转换规范指南

## ⚠️ 重要：类型转换是硬性要求

**函数详情中的 `data.type` 字段明确说明了提交时应该使用的类型，不符合类型会导致后端解析失败！**

## 核心原则

1. **所有类型转换必须使用统一工具函数**
   - 不要自己实现类型转换逻辑
   - 使用 `typeConverter.ts` 中的工具函数

2. **必须根据 `field.data.type` 进行转换**
   - 函数详情中的 `request` 字段数组包含每个字段的 `data.type`
   - 这是后端期望的类型，必须严格遵守

3. **类型转换的四个关键场景**
   - URL 参数初始化：URL 参数都是字符串，需要转换
   - 提交数据：提交时需要根据类型转换
   - 回调接口的 request 参数：需要根据字段类型转换
   - 组件显示：需要正确匹配类型（数字 vs 字符串）

## 统一工具函数

### 1. `convertValueByFieldType(value, field)`

根据字段配置转换单个值。

```typescript
import { convertValueByFieldType } from '../utils/typeConverter'

// 基础类型
convertValueByFieldType('1', { data: { type: 'int' } })  // 1
convertValueByFieldType('1.5', { data: { type: 'float' } })  // 1.5
convertValueByFieldType('true', { data: { type: 'bool' } })  // true

// 数组类型
convertValueByFieldType('1,2,3', { data: { type: '[]int' } })  // [1, 2, 3]
convertValueByFieldType(['1', '2'], { data: { type: '[]int' } })  // [1, 2]
```

### 2. `convertFormDataToRequestByType(formData, functionDetail)`

将表单数据转换为请求格式，并根据字段类型进行转换。

**这是提交数据和回调接口 request 参数转换的统一函数！**

```typescript
import { convertFormDataToRequestByType } from '../utils/typeConverter'

// 在回调接口中使用
const submitData = props.formRenderer.getSubmitData()
const functionDetail = props.formRenderer.getFunctionDetail?.()
const requestData = convertFormDataToRequestByType(submitData, functionDetail || {})

// 在初始化器中使用
const requestData = convertFormDataToRequestByType(allFormData, functionDetail)
```

## 使用场景

### 场景 1：URL 参数初始化

URL 参数都是字符串，需要根据 `field.data.type` 转换。

```typescript
// ✅ 正确：使用 convertBasicType 或 convertArrayType
const fieldType = field.data?.type || DataType.STRING
const convertedRaw = convertBasicType(originalValue, fieldType)
```

### 场景 2：提交数据

提交时需要根据 `field.data.type` 转换。

```typescript
// ✅ 正确：使用 convertFormDataToRequestByType
const submitData = formDataStore.getSubmitData(requestFields)
// 提交时已经根据类型转换，无需额外处理
```

### 场景 3：回调接口的 request 参数

回调接口的 `request` 参数需要根据字段类型转换。

```typescript
// ✅ 正确：使用 convertFormDataToRequestByType
const submitData = props.formRenderer.getSubmitData()
const functionDetail = props.formRenderer.getFunctionDetail?.()
const requestData = convertFormDataToRequestByType(submitData, functionDetail || {})

const requestBody = {
  code: props.field.code,
  type: queryType,
  value: queryValue,
  request: requestData,  // 🔥 使用转换后的数据
  value_type: props.field.data?.type
}
```

### 场景 4：组件显示

组件显示时需要正确匹配类型（数字 vs 字符串）。

```typescript
// ✅ 正确：使用转换后的值匹配选项
const option = options.value.find((opt: any) => {
  // 支持多种类型比较
  return opt.value === processedValue.raw || 
         String(opt.value) === String(processedValue.raw)
})
```

## 常见错误

### ❌ 错误 1：直接使用字符串值

```typescript
// ❌ 错误：直接使用字符串，没有转换
const requestData = {
  topic_id: '1',  // 应该是数字 1
  option_ids: '1'  // 应该是数组 [1]
}
```

### ❌ 错误 2：自己实现类型转换

```typescript
// ❌ 错误：自己实现类型转换逻辑
if (fieldType === 'int') {
  request[key] = parseInt(rawValue, 10)
} else if (fieldType === '[]int') {
  request[key] = rawValue.split(',').map(v => parseInt(v, 10))
}
```

### ❌ 错误 3：忘记转换回调接口的 request 参数

```typescript
// ❌ 错误：直接使用 getSubmitData()，没有转换
const requestBody = {
  code: props.field.code,
  request: props.formRenderer.getSubmitData()  // 值都是字符串！
}
```

## ✅ 正确做法

### 1. 使用统一工具函数

```typescript
// ✅ 正确：使用 convertFormDataToRequestByType
import { convertFormDataToRequestByType } from '../utils/typeConverter'

const submitData = props.formRenderer.getSubmitData()
const functionDetail = props.formRenderer.getFunctionDetail?.()
const requestData = convertFormDataToRequestByType(submitData, functionDetail || {})
```

### 2. 确保 functionDetail 已准备好

```typescript
// ✅ 正确：检查 functionDetail 是否已准备好
import { getFormRequestFields } from '@/utils/functionSchemaSelectors'

const functionDetail = props.formRenderer?.getFunctionDetail?.()
if (!functionDetail || getFormRequestFields(functionDetail).length === 0) {
  // functionDetail 还没准备好，等待下次触发
  return
}
```

### 3. 在初始化器中使用转换后的值

```typescript
// ✅ 正确：使用转换后的值作为 raw
const initializedValue = createFieldValue(
  field,
  convertedValue,  // 🔥 使用转换后的值，而不是 currentValue.raw
  display,
  meta
)
```

## 检查清单

在实现或修改组件时，确保：

- [ ] URL 参数初始化时，使用 `convertBasicType` 或 `convertArrayType` 转换
- [ ] 提交数据时，使用 `convertFormDataToRequestByType` 转换
- [ ] 回调接口的 request 参数，使用 `convertFormDataToRequestByType` 转换
- [ ] 组件显示时，使用转换后的值匹配选项
- [ ] 初始化器返回的值，使用转换后的值作为 `raw`
- [ ] 所有类型转换都使用统一工具函数，不自己实现
- [ ] 选项映射构建使用 `buildOptionMaps`，查找使用 `getOptionLabelFromMap`

## 最新优化记录

### 2025-01-XX：统一类型转换和选项映射

#### 1. 统一数组类型转换
- **问题**：`MultiSelectWidget.vue` 中有硬编码的 `parseInt`/`parseFloat` 逻辑
- **解决**：统一使用 `convertArrayType` 工具函数
- **影响文件**：
  - `web/src/core/widgets-v2/components/MultiSelectWidget.vue`

#### 2. 提取选项映射工具函数
- **问题**：`MultiSelectWidgetInitializer.ts` 中有重复的选项映射构建和查找逻辑
- **解决**：新增 `buildOptionMaps` 和 `getOptionLabelFromMap` 工具函数
- **影响文件**：
  - `web/src/core/widgets-v2/utils/typeConverter.ts`（新增函数）
  - `web/src/core/widgets-v2/initializers/MultiSelectWidgetInitializer.ts`（使用新函数）

#### 3. 修复 ChartRenderer 类型转换
- **问题**：`ChartRenderer.vue` 中使用硬编码的类型转换逻辑
- **解决**：使用 `convertValueByFieldType` 统一工具函数
- **影响文件**：
  - `web/src/components/ChartRenderer.vue`

## 相关文件

- `web/src/core/widgets-v2/utils/typeConverter.ts` - 类型转换工具函数
- `web/src/core/widgets-v2/utils/valueConverter.ts` - 值类型转换工具
- `web/src/core/widgets-v2/initializers/SelectWidgetInitializer.ts` - SelectWidget 初始化器示例
- `web/src/core/widgets-v2/initializers/MultiSelectWidgetInitializer.ts` - MultiSelectWidget 初始化器示例

## 总结

**记住：类型转换是硬性要求，必须使用统一工具函数，确保所有字段都根据 `field.data.type` 正确转换！**
