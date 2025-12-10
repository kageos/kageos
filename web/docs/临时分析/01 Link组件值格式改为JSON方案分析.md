# Link 组件值格式改为 JSON 方案分析

## 📋 当前情况

### 当前格式
```
"[查看会议室详情]/luobei/demo/crm/meeting_room_list?eq=id%3A2"
```

### 问题
- 无法直接知道目标函数的类型（table 或 form）
- 前端需要通过 Tab 或详情来判断函数类型，才能决定是否保留 table 参数
- 字符串解析容易出错（需要正则匹配）

## 🎯 目标格式

### 新格式（JSON）
```json
{
  "type": "table",
  "name": "查看会议室详情",
  "url": "/luobei/demo/crm/meeting_room_list?eq=id%3A2"
}
```

### 优势
1. **直接获取函数类型**：不需要通过 Tab 或详情判断
2. **更准确处理参数**：根据 `type` 字段决定是否保留 table 参数
3. **数据结构清晰**：易于扩展（未来可以添加更多字段）
4. **类型安全**：JSON 解析比字符串正则更可靠

## 🔍 后端修改分析

### 修改位置
**文件**：`sdk/agent-app/app/url_builder.go`

**函数**：`BuildFunctionUrlWithText`

### 当前实现
```go
// 第 58-59 行：已经可以获取函数类型
switch template.TemplateType() {
case TemplateTypeTable:
    // ...
case TemplateTypeForm:
    // ...
}

// 第 132-133 行：返回字符串格式
if text != "" {
    return fmt.Sprintf("[%s]%s", text, finalUrl), nil
}
```

### 修改方案

#### 方案 1：直接返回 JSON 字符串（推荐）
```go
// 定义结构体
type LinkValue struct {
    Type string `json:"type"` // "table" 或 "form"
    Name string `json:"name"` // 链接文本
    URL  string `json:"url"`  // 链接 URL
}

// 修改返回逻辑
func (ctx *Context) BuildFunctionUrlWithText(...) (string, error) {
    // ... 现有逻辑 ...
    
    templateType := template.TemplateType()
    linkValue := LinkValue{
        Type: string(templateType), // "table" 或 "form"
        Name: text,
        URL:  finalUrl,
    }
    
    // 返回 JSON 字符串
    jsonBytes, err := json.Marshal(linkValue)
    if err != nil {
        return "", fmt.Errorf("序列化 link 值失败: %w", err)
    }
    return string(jsonBytes), nil
}
```

**优点**：
- 修改最小，只改返回值格式
- 保持函数签名不变（仍然返回 string）
- 向后兼容：前端可以同时支持旧格式和新格式

**缺点**：
- 返回值是 JSON 字符串，需要前端解析

#### 方案 2：返回结构体（不推荐）
需要修改函数签名，影响面太大。

### 修改难度评估

**难度**：⭐⭐（简单）

**原因**：
1. ✅ 后端已经有 `template.TemplateType()` 可以获取函数类型
2. ✅ 只需要修改返回格式，不需要改变业务逻辑
3. ✅ 可以保持向后兼容（前端同时支持旧格式和新格式）

### 需要修改的文件

1. **后端**：
   - `sdk/agent-app/app/url_builder.go`：修改 `BuildFunctionUrlWithText` 函数
   - 可能需要添加 `LinkValue` 结构体定义

2. **前端**：
   - `web/src/core/widgets-v2/components/LinkWidget.vue`：修改解析逻辑
   - 支持同时解析旧格式（字符串）和新格式（JSON）

## 🎨 前端修改方案

### 当前解析逻辑
```typescript
// LinkWidget.vue
const parsedLink = computed(() => {
  const url = props.value?.raw || ''
  if (!url) return { text: '', url: '' }
  
  // 解析 "[text]url" 格式
  const match = url.match(/^\[([^\]]+)\](.+)$/)
  if (match) {
    return {
      text: match[1],
      url: match[2]
    }
  }
  
  return { text: '', url: url }
})
```

### 新解析逻辑（兼容旧格式）
```typescript
interface LinkValue {
  type?: 'table' | 'form'  // 可选，兼容旧格式
  name?: string
  url: string
}

const parsedLink = computed(() => {
  const raw = props.value?.raw || ''
  if (!raw) return { text: '', url: '', type: undefined }
  
  // 尝试解析 JSON 格式
  try {
    const jsonValue = JSON.parse(raw)
    if (jsonValue && typeof jsonValue === 'object' && jsonValue.url) {
      return {
        text: jsonValue.name || '',
        url: jsonValue.url,
        type: jsonValue.type  // 'table' 或 'form'
      }
    }
  } catch {
    // 不是 JSON，继续解析旧格式
  }
  
  // 解析旧格式 "[text]url"
  const match = raw.match(/^\[([^\]]+)\](.+)$/)
  if (match) {
    return {
      text: match[1],
      url: match[2],
      type: undefined  // 旧格式没有类型信息
    }
  }
  
  // 纯 URL
  return {
    text: '',
    url: raw,
    type: undefined
  }
})
```

### 使用 type 字段优化跳转逻辑

在 `WorkspaceView.vue` 的 `handleNodeClick` 中：

```typescript
// 从 link 值中获取函数类型
const linkValue = parsedLink.value
if (linkValue.type) {
  // 直接使用 link 值中的类型，不需要通过 Tab 判断
  const preservedQuery = linkValue.type === 'table'
    ? preserveQueryParamsForTable(route.query)
    : preserveQueryParamsForForm(route.query)
  
  router.replace({ path: targetPath, query: preservedQuery }).catch(() => {})
} else {
  // 旧格式或没有类型信息，使用现有逻辑（通过 Tab 判断）
  // ...
}
```

## 📊 影响分析

### 向后兼容性
✅ **完全兼容**

- 前端可以同时支持旧格式（字符串）和新格式（JSON）
- 旧数据仍然可以正常显示和跳转
- 新数据可以使用新格式，获得更好的体验

### 性能影响
- **无影响**：JSON 解析性能与字符串正则匹配相当
- 甚至可能更快（JSON 解析是原生操作）

### 数据迁移
- **不需要迁移**：旧数据可以继续使用
- 新数据自动使用新格式

## ✅ 结论

### 推荐实施
**理由**：
1. ✅ 后端修改简单，只需要改返回值格式
2. ✅ 前端可以完全向后兼容
3. ✅ 可以解决当前的问题（link 跳转到 form 函数时自动添加 table 参数）
4. ✅ 数据结构更清晰，易于扩展

### 实施步骤
1. **后端**：修改 `BuildFunctionUrlWithText` 返回 JSON 格式
2. **前端**：修改 `LinkWidget.vue` 支持解析 JSON 格式（兼容旧格式）
3. **前端**：在 `handleNodeClick` 中使用 `type` 字段优化参数保留逻辑
4. **测试**：确保旧格式和新格式都能正常工作

### 风险评估
- **低风险**：完全向后兼容，不影响现有功能
- **收益高**：解决当前问题，提升代码质量

