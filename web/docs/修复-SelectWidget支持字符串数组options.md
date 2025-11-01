# 修复：SelectWidget 支持字符串数组 options

## 🐛 问题描述

用户反馈 Select 组件无法渲染选项。

### 后端返回的字段配置

```json
{
  "code": "priority",
  "name": "优先级",
  "validation": "required,oneof=低,中,高",
  "widget": {
    "type": "select",
    "config": {
      "creatable": false,
      "default": "中",
      "options": ["低", "中", "高"],  // ❌ 简单字符串数组
      "placeholder": ""
    }
  }
}
```

### 问题根因

`SelectWidget.initOptions()` 方法期望 `options` 是 `SelectOption[]` 类型：

```typescript
// SelectOption 接口定义
export interface SelectOption {
  label: string
  value: any
  disabled?: boolean
}
```

但后端返回的是简单字符串数组：`["低", "中", "高"]`

---

## 🔧 解决方案

### 修改 `SelectWidget.initOptions()`

```typescript
/**
 * 初始化选项
 */
private initOptions(): void {
  // 从配置中获取初始选项（如果有）
  const initialOptions = this.selectConfig.options
  if (initialOptions && Array.isArray(initialOptions) && initialOptions.length > 0) {
    // 🔥 兼容两种格式：
    // 1. 字符串数组：["低", "中", "高"]
    // 2. 对象数组：[{ label: "低", value: "低" }]
    if (typeof initialOptions[0] === 'string') {
      // 字符串数组 -> SelectOption[]
      this.options.value = (initialOptions as string[]).map(opt => ({
        label: opt,
        value: opt
      }))
    } else {
      // 已经是 SelectOption[] 格式
      this.options.value = initialOptions as SelectOption[]
    }
    
    console.log(`[SelectWidget] ${this.field.code} 初始化选项:`, this.options.value)
  }
  
  // 如果有初始值，触发一次搜索获取 displayInfo
  const currentValue = this.formManager.getValue(this.fieldPath)
  if (currentValue?.raw !== null && currentValue?.raw !== undefined) {
    this.handleSearch('', true) // 静默搜索（by_field_values）
  }
}
```

---

## ✅ 修复效果

### 修复前
```typescript
// 后端返回
options: ["低", "中", "高"]

// SelectWidget 直接使用，无法渲染
this.options.value = ["低", "中", "高"]  // ❌ 类型不匹配
```

### 修复后
```typescript
// 后端返回
options: ["低", "中", "高"]

// SelectWidget 自动转换
this.options.value = [
  { label: "低", value: "低" },
  { label: "中", value: "中" },
  { label: "高", value: "高" }
]  // ✅ 正确的 SelectOption[] 格式
```

---

## 🎯 支持的两种格式

### 格式 1：简单字符串数组（常用）
```json
{
  "widget": {
    "type": "select",
    "config": {
      "options": ["选项1", "选项2", "选项3"]
    }
  }
}
```

转换后：
```typescript
[
  { label: "选项1", value: "选项1" },
  { label: "选项2", value: "选项2" },
  { label: "选项3", value: "选项3" }
]
```

### 格式 2：对象数组（高级用法）
```json
{
  "widget": {
    "type": "select",
    "config": {
      "options": [
        { "label": "选项1", "value": 1, "disabled": false },
        { "label": "选项2", "value": 2, "disabled": true },
        { "label": "选项3", "value": 3 }
      ]
    }
  }
}
```

直接使用，无需转换。

---

## 🧪 测试场景

### 测试用例 1：优先级字段（字符串数组）
```json
{
  "code": "priority",
  "name": "优先级",
  "widget": {
    "type": "select",
    "config": {
      "default": "中",
      "options": ["低", "中", "高"]
    }
  }
}
```

预期效果：
- ✅ 下拉框显示：低、中、高
- ✅ 默认选中：中
- ✅ 提交时 `value` 为 "中"

### 测试用例 2：工单状态（字符串数组）
```json
{
  "code": "status",
  "name": "工单状态",
  "widget": {
    "type": "select",
    "config": {
      "default": "待处理",
      "options": ["待处理", "处理中", "已完成", "已关闭"]
    }
  }
}
```

预期效果：
- ✅ 下拉框显示所有状态
- ✅ 默认选中：待处理

### 测试用例 3：对象数组（label ≠ value）
```json
{
  "code": "user_id",
  "name": "负责人",
  "widget": {
    "type": "select",
    "config": {
      "options": [
        { "label": "张三", "value": 1 },
        { "label": "李四", "value": 2 },
        { "label": "王五", "value": 3 }
      ]
    }
  }
}
```

预期效果：
- ✅ 下拉框显示：张三、李四、王五
- ✅ 提交时 `value` 为数字 ID（1、2、3）

---

## 🔍 代码审查要点

### 1. 空数组检查
```typescript
if (initialOptions && Array.isArray(initialOptions) && initialOptions.length > 0) {
  // ✅ 确保数组不为空，避免 initialOptions[0] 为 undefined
}
```

### 2. 类型转换
```typescript
if (typeof initialOptions[0] === 'string') {
  // ✅ 检测第一个元素的类型来判断整个数组的格式
  this.options.value = (initialOptions as string[]).map(opt => ({
    label: opt,
    value: opt
  }))
}
```

### 3. 类型安全
```typescript
// 修复前：隐式 any 类型
.map(option => ...)  // ❌ TypeScript 报错

// 修复后：显式类型声明
.map((option: SelectOption) => ...)  // ✅ 类型安全
```

---

## 📝 总结

本次修复让 `SelectWidget` 兼容两种 `options` 格式：
1. **简单字符串数组**：`["选项1", "选项2"]`（label = value）
2. **对象数组**：`[{ label: "...", value: ... }]`（label ≠ value）

这样既满足了简单场景的易用性，也保留了高级场景的灵活性。

### 关键改进
- ✅ 自动检测 `options` 格式
- ✅ 自动转换字符串数组为 `SelectOption[]`
- ✅ 保持对象数组的原有功能
- ✅ 添加类型安全检查
- ✅ 添加调试日志

---

## 🚀 后续优化建议

### 1. 支持数字数组
```typescript
options: [1, 2, 3, 4, 5]  // 评分场景
// 转换为
[
  { label: "1", value: 1 },
  { label: "2", value: 2 },
  ...
]
```

### 2. 支持键值对对象
```typescript
options: {
  "low": "低优先级",
  "medium": "中优先级",
  "high": "高优先级"
}
// 转换为
[
  { label: "低优先级", value: "low" },
  { label: "中优先级", value: "medium" },
  { label: "高优先级", value: "high" }
]
```

### 3. 支持分组
```typescript
options: [
  {
    label: "水果",
    options: [
      { label: "苹果", value: "apple" },
      { label: "香蕉", value: "banana" }
    ]
  },
  {
    label: "蔬菜",
    options: [
      { label: "白菜", value: "cabbage" }
    ]
  }
]
```

这些优化可以根据实际需求逐步添加。目前的实现已经满足常见场景。🎉

