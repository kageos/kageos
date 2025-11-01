# List 内 Select 渲染测试说明

## 🎯 实现目标

实现了**最复杂**的表单渲染场景：**List 内 Select**（收银台场景）

这个场景包含：
- ✅ 递归渲染（List 渲染子 Widget）
- ✅ 事件驱动通信（Select → List → FormManager）
- ✅ 回调处理（OnSelectFuzzy）
- ✅ 动态添加/删除行
- ✅ 聚合统计配置

## 📁 新增文件

### 1. **SelectWidget.ts**
- 下拉选择组件
- 支持远程搜索（remote）
- 支持回调（OnSelectFuzzy）
- 支持 displayInfo 保存
- 发送 `field:search` 事件

### 2. **ListWidget.ts**
- 列表容器组件
- 支持动态添加/删除行
- 订阅子组件事件（`field:search`, `field:change`）
- 支持聚合统计（TODO: 需要 ExpressionParser）
- 递归渲染子 Widget

### 3. **测试数据**
- 简单表单（Test 1）
- 工单表单（Test 2）
- 🔥 收银台场景 - List 内 Select（Test 3）

## 🏗️ 架构亮点

### 1. **事件驱动**
```
Select 组件
  └─ 用户搜索
      └─ emit('field:search', eventData)  // 发送事件
          └─ FormDataManager（事件总线）
              └─ List 组件（订阅者）
                  └─ handleChildSearch()  // 处理搜索
                      └─ 调用后端 API
```

### 2. **递归渲染**
```
FormRenderer
  └─ renderField(List)
      └─ ListWidget.render()
          └─ renderItem(0)
              └─ WidgetFactory.createWidget(Select)
                  └─ SelectWidget.render()
          └─ renderItem(1)
              └─ WidgetFactory.createWidget(Select)
```

### 3. **Widget 注册机制**
```
FormRenderer
  ├─ allWidgets Map
  │   ├─ "products" → ListWidget
  │   ├─ "products[0].product_id" → SelectWidget
  │   ├─ "products[0].quantity" → InputWidget
  │   ├─ "products[1].product_id" → SelectWidget
  │   └─ "products[1].quantity" → InputWidget
  └─ captureSnapshot()  // 遍历所有 Widget
```

### 4. **数据结构**
```typescript
// FormDataManager 中的数据
{
  "products[0].product_id": {
    raw: 1,
    display: "商品 A",
    meta: {
      displayInfo: { label: "商品 A", value: 1, price: 100 }
    }
  },
  "products[0].quantity": {
    raw: 2,
    display: "2",
    meta: {}
  },
  "products[1].product_id": {
    raw: 3,
    display: "商品 B",
    meta: {
      displayInfo: { label: "商品 B", value: 3, price: 200 }
    }
  },
  "products[1].quantity": {
    raw: 1,
    display: "1",
    meta: {}
  }
}
```

## 🧪 测试步骤

### 1. 访问测试页面
```
http://localhost:5173/test/form-renderer
```

### 2. 切换到收银台测试
点击"切换测试数据"按钮 2 次，切换到"收银台场景 - List 内 Select"

### 3. 测试功能

#### a. 查看初始状态
- 看到"客户姓名"输入框
- 看到"商品列表"，默认有 2 行
- 每行有：行号、商品下拉框、数量输入框、删除按钮

#### b. 测试添加行
- 点击"添加一行"按钮
- 应该添加第 3 行

#### c. 测试删除行
- 点击某一行的"删除"按钮
- 该行应该被移除

#### d. 测试 Select 搜索
- 点击商品下拉框
- 应该触发远程搜索
- 查看控制台，应该看到：
  ```
  [SelectWidget] 发送搜索事件: {...}
  [ListWidget] 收到子组件搜索事件: {...}
  [ListWidget] 处理子组件搜索: {...}
  ```

#### e. 测试提交
- 填写客户姓名
- 选择商品（每一行）
- 填写数量（每一行）
- 点击"提交"
- 查看提交结果，应该包含：
  ```json
  {
    "customer_name": "张三",
    "products": [
      {
        "product_id": 1,
        "quantity": 2
      },
      {
        "product_id": 3,
        "quantity": 1
      }
    ]
  }
  ```

#### f. 测试分享（快照）
- 点击"分享"按钮
- 查看快照数据，应该包含所有行的 Widget 快照：
  ```json
  {
    "view_id": "test_xxxxx",
    "function_code": "cashier_desk",
    "widget_snapshots": [
      {
        "field_path": "customer_name",
        "field_code": "customer_name",
        "widget_type": "input",
        "field_value": {...}
      },
      {
        "field_path": "products[0].product_id",
        "field_code": "product_id",
        "widget_type": "select",
        "field_value": {...},
        "component_data": {
          "options": [...],
          "loading": false
        }
      },
      ...
    ]
  }
  ```

## 📊 控制台日志

正常情况下，控制台应该显示：

### 初始化阶段
```
[WidgetFactory] 初始化，已注册 Widget: ['input', 'text', 'textarea', 'select', 'list']
[SimpleFormRenderer] 初始化表单
[SimpleFormRenderer] 注册 Widget: customer_name
[SimpleFormRenderer] 注册 Widget: products
[ListWidget] 添加行 0
[ListWidget] 添加行 1
[SimpleFormRenderer] 注册 Widget: products[0].product_id
[SimpleFormRenderer] 注册 Widget: products[0].quantity
[SimpleFormRenderer] 注册 Widget: products[1].product_id
[SimpleFormRenderer] 注册 Widget: products[1].quantity
```

### 搜索阶段（点击 Select）
```
[SelectWidget] 发送搜索事件: {field_path: "products[0].product_id", ...}
[ListWidget] 收到子组件搜索事件: {...}
[ListWidget] 处理子组件搜索: {...}
[ListWidget] 搜索完成，更新子组件选项
```

### 添加行阶段
```
[ListWidget] 添加行 2
[SimpleFormRenderer] 注册 Widget: products[2].product_id
[SimpleFormRenderer] 注册 Widget: products[2].quantity
```

### 删除行阶段
```
[ListWidget] 删除行 1
[SimpleFormRenderer] 注销 Widget: products[1].product_id
[SimpleFormRenderer] 注销 Widget: products[1].quantity
[ListWidget] 重新计算聚合
```

## 🔧 TODO（未来扩展）

### 1. 回调 API 集成
目前 `SelectWidget.handleSearch()` 只是模拟数据，需要：
- 调用实际的 `/api/v1/callback/.../...?_type=OnSelectFuzzy` API
- 解析响应，更新 `options`
- 将 `displayInfo` 保存到 `FieldValue.meta`

### 2. ExpressionParser
目前 `ListWidget.recalculateAggregation()` 只是空实现，需要：
- 实现表达式解析器
- 支持 `sum(product_id.price, *quantity)` 语法
- 计算聚合结果并显示

### 3. 表单验证
需要集成验证逻辑：
- List 的 `min=1` 验证
- Select 的 `required` 验证
- Input 的 `min=1` 验证

### 4. 更多组件
- MultiSelectWidget（多选）
- NumberWidget（数字输入，支持 spinner）
- DateWidget（日期选择）
- StructWidget（结构体，如果需要）

## 🎉 总结

这次实现了：
1. ✅ **组件化架构** - 避免屎山，每个组件职责单一
2. ✅ **事件驱动通信** - 解耦合，Select 不直接调用 List
3. ✅ **递归渲染** - 支持无限嵌套
4. ✅ **Widget 注册机制** - 支持快照和分享
5. ✅ **OOP 设计** - BaseWidget 基类，扩展方便

**最重要的是**：架构清晰，后续添加新组件不需要改动现有代码！🚀

## 🔗 相关文档
- `web/docs/新版本架构设计.md`
- `web/docs/架构设计-事件驱动与组件职责.md`
- `web/docs/架构设计-收银台完整场景示例.md`
- `web/docs/架构设计-组件快照机制.md`
- `web/composables/README.md`

