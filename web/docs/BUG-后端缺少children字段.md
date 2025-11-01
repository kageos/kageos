# BUG：后端 product_quantities 缺少 children 字段

## 问题描述

收银台功能的 `product_quantities` 字段无法渲染，因为后端返回的数据中**缺少 `children` 字段**。

## 后端返回的数据（错误）

```json
{
  "code": "product_quantities",
  "data": {"type": "[]struct"},
  "widget": {"type": "table"},
  "validation": "required,min=1",
  "callbacks": null  // ← 这里也是 null，但应该是 []
}
```

**❌ 缺少 `children` 字段！**

## 应该返回的数据（正确）

```json
{
  "code": "product_quantities",
  "name": "商品清单",
  "data": {"type": "[]struct"},
  "widget": {"type": "table", "config": null},
  "validation": "required,min=1",
  "callbacks": null,
  "children": [  // ← 应该有这个字段！
    {
      "code": "product_id",
      "name": "商品",
      "type": "int",
      "data": {"type": "int"},
      "validation": "required",
      "callbacks": ["OnSelectFuzzy"],  // ← 子字段的 callbacks
      "widget": {
        "type": "select",
        "config": {
          "placeholder": "请选择商品",
          "creatable": false
        }
      }
    },
    {
      "code": "quantity",
      "name": "数量",
      "type": "int",
      "data": {"type": "int"},
      "validation": "required,min=1",
      "widget": {
        "type": "input",
        "config": {
          "placeholder": "请输入数量"
        }
      }
    }
  ]
}
```

## 问题原因分析

### 1. 后端解析逻辑存在
后端 `decode.go` 中有正确的解析逻辑：

```go
// parseNestedStructOrSlice 递归解析嵌套的结构体或切片
func parseNestedStructOrSlice(fieldType reflect.Type, parentTags *FieldTags) error {
    // 只有明确指定为 table 时，才解析切片中的结构体
    if widgetType == TypeTable {
        if fieldType.Kind() == reflect.Slice {
            elemType := fieldType.Elem()

            // 如果切片元素是结构体，递归解析
            if elemType.Kind() == reflect.Struct {
                children, err := parseStructFields(elemType)
                if err != nil {
                    return fmt.Errorf("failed to parse slice element struct: %w", err)
                }
                parentTags.Children = children  // ← 这里设置了 Children
            }
        }
    }
    return nil
}

// ConvertTagsToField 转换时也会递归处理
func ConvertTagsToField(tags *FieldTags) *Field {
    // ...
    // 递归转换Children字段
    if len(tags.Children) > 0 {
        field.Children = make([]*Field, 0, len(tags.Children))
        for _, childTags := range tags.Children {
            childField := ConvertTagsToField(childTags)
            field.Children = append(field.Children, childField)
        }
    }
    return field
}
```

### 2. 可能的问题点

**猜测1：序列化问题**
- `FieldTags.Children` 被正确设置了
- `ConvertTagsToField` 也正确转换了
- 但是在序列化为 JSON 返回给前端时，`children` 字段被过滤掉了

**猜测2：缓存问题**
- 后端可能有缓存，使用的是旧版本的数据
- 需要清除缓存或重启服务

**猜测3：解析条件不满足**
- `CashierProductQuantity` 没有被正确识别为结构体
- 或者 `widget:"name:商品清单;type:table"` 没有被正确解析

### 3. Callbacks 问题

后端的 `DecodeForm` 只处理顶层字段的 callbacks：

```go
field := ConvertTagsToField(fieldTags)
calls, ok := fieldsCallback[field.Code]  // 只查找 "product_quantities"
if ok {
    field.Callbacks = calls
}
```

但是，`OnSelectFuzzyMap` 中的 key 是 `"product_id"`，不是 `"product_quantities"`：

```go
OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
    "product_id": onSelectFuzzyProduct,  // ← 这是嵌套字段
    "member_id":  onSelectFuzzyMember,   // ← 这是顶层字段
},
```

所以嵌套字段的 callbacks 没有被正确设置！

## 解决方案

### 方案1：修复后端（推荐）

修改 `DecodeForm` 函数，递归设置嵌套字段的 callbacks：

```go
// 新增：递归设置嵌套字段的 callbacks
func setNestedCallbacks(field *Field, fieldsCallback map[string][]string, prefix string) {
    // 设置当前字段的 callbacks
    fieldPath := prefix + field.Code
    if calls, ok := fieldsCallback[field.Code]; ok {
        field.Callbacks = calls
    }
    
    // 递归处理子字段
    if len(field.Children) > 0 {
        for _, childField := range field.Children {
            setNestedCallbacks(childField, fieldsCallback, fieldPath+".")
        }
    }
}

// 修改 DecodeForm
func DecodeForm(fieldsCallback map[string][]string, request, response interface{}) (requestFields []*Field, responseFields []*Field, err error) {
    // ...
    for _, fieldTags := range requestResult.Tags {
        field := ConvertTagsToField(fieldTags)
        
        // 🔥 新增：递归设置 callbacks（包括嵌套字段）
        setNestedCallbacks(field, fieldsCallback, "")
        
        requestFields = append(requestFields, field)
    }
    // ...
}
```

### 方案2：调试后端

1. 在 `parseNestedStructOrSlice` 中添加日志：
```go
fmt.Printf("[DEBUG] parseNestedStructOrSlice: fieldType=%v, widgetType=%s\n", fieldType, widgetType)
if len(parentTags.Children) > 0 {
    fmt.Printf("[DEBUG] Children count: %d\n", len(parentTags.Children))
}
```

2. 在 `ConvertTagsToField` 中添加日志：
```go
if len(tags.Children) > 0 {
    fmt.Printf("[DEBUG] ConvertTagsToField: %s has %d children\n", tags.Json, len(tags.Children))
}
```

3. 检查序列化输出：
```go
jsonData, _ := json.MarshalIndent(field, "", "  ")
fmt.Printf("[DEBUG] Field JSON:\n%s\n", string(jsonData))
```

## 前端临时解决方案

前端已经做了以下修复，**只要后端返回正确的 `children` 字段，就能正常渲染**：

1. ✅ 将 `FieldConfig.properties` 改为 `children`（对齐后端）
2. ✅ `ListWidget` 使用 `field.children` 而不是 `field.properties`
3. ✅ `WidgetFactory` 注册 `"table"` 为 `ListWidget` 的别名
4. ✅ 添加测试数据 Test 3，模拟正确的后端响应

## 测试步骤

### 1. 前端测试（测试数据）
访问 `http://localhost:5173/test/form-renderer`，点击"切换测试数据"2次，查看"收银台场景"。

如果渲染正常（显示商品清单列表，带添加/删除按钮），说明前端逻辑正确。

### 2. 实际功能测试
访问 `http://localhost:5173/workspace/luobei/wal/tools/cashier_desk`。

**期望**：看到表单，包括：
- 客户姓名输入框
- 商品清单列表（带添加/删除按钮）
- 每行有：商品选择框 + 数量输入框
- 会员卡选择框
- 备注文本框

**实际**：如果商品清单无法渲染，说明后端返回的数据缺少 `children` 字段。

## 总结

- **前端已完成**：✅ 架构设计完善，代码已对齐后端
- **后端需修复**：❌ `children` 字段缺失，`callbacks` 设置不完整
- **优先级**：🔥 高（阻塞核心功能）

修复后端的 `children` 字段问题后，前端即可正常渲染 List 内 Select 场景！

