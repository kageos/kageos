# 架构设计 - URL 回想与缓存策略

> ⚠️ **重要提示**：本方案已被 **分享视图系统** 取代。  
> 分享视图方案更优雅、更强大，支持富文本、大数据、聚合信息，且只需一次请求。  
> 详见：`架构设计-分享视图系统.md`  
> 
> 本文档保留作为备选方案参考。

---

## 核心问题

### 场景描述

用户在表单中通过模糊搜索选择了商品，表单保存了数据后，将 URL 分享给其他人。新用户打开链接时，只有字段的 **原始值**（如 `product_id: 1`），但缺少 **显示信息**（如商品名称、价格等 `displayInfo`）。

### 问题分析

```typescript
// 用户 A 的表单数据（完整）
{
  product_id: {
    raw: 1,
    display: "iPhone 15 Pro - ¥7999",
    meta: {
      displayInfo: {
        商品名称: "iPhone 15 Pro",
        价格: 7999,
        库存: 50
      }
    }
  }
}

// URL 分享后，用户 B 收到的数据（只有值）
{
  product_id: 1  // ❌ 缺少 display 和 displayInfo
}
```

**为什么会丢失信息？**
- URL 只能传递简单的 key-value 数据
- `displayInfo` 数据量大，不适合放在 URL 中
- 用户 B 的浏览器没有用户 A 的本地缓存

---

## 旧版本的复杂方案 ❌

### 方案描述

List 内多个 Select，统一收集所有 Select 的值，一次性回想。

```typescript
// 场景：List 内 10 行，每行一个 Select
products: [
  { product_id: 1, quantity: 2 },
  { product_id: 1, quantity: 1 },  // 重复值
  { product_id: 4, quantity: 3 },
  { product_id: 1, quantity: 1 },  // 重复值
  // ... 共 10 行
]

// 旧版本：ListWidget 统一收集
const allProductIds = [1, 1, 4, 1, ...]  // 收集所有值
const uniqueIds = [...new Set(allProductIds)]  // 去重: [1, 4]

// 一次性回想
await callOnSelectFuzzy({
  code: "product_id",
  value: uniqueIds,  // [1, 4]
  input_type: "by_field_values",  // 🔥 标识：根据值回想
  request: formData
})

// 后端 SQL: SELECT * FROM products WHERE product_id IN (1, 4)

// 然后 ListWidget 需要把结果分发到各个子 Select
for (const [index, row] of products.entries()) {
  const productId = row.product_id
  const displayInfo = resultMap[productId]  // 从回想结果中找到对应的信息
  
  // 🔥 ListWidget 需要操作子组件的值
  selectWidget[index].setValue({
    raw: productId,
    display: displayInfo.label,
    meta: { displayInfo: displayInfo.displayInfo }
  })
}
```

### 问题分析

| 问题 | 说明 | 严重度 |
|------|------|-------|
| **耦合度高** | ListWidget 需要操作子组件的值，违反"各自管各自"原则 | ⭐⭐⭐⭐⭐ |
| **复杂度高** | 需要收集、去重、回想、分发，逻辑链路长 | ⭐⭐⭐⭐ |
| **可维护性差** | 新增组件类型需要修改 ListWidget | ⭐⭐⭐⭐ |
| **调试困难** | 值的流转不清晰，难以追踪问题 | ⭐⭐⭐ |

---

## 新版本简化方案 ✅

### 核心思路

> **各自管各自 + 缓存复用**

1. **各自管各自**：每个 Select 独立回想，不依赖 ListWidget
2. **缓存层**：相同值的回调自动复用缓存结果
3. **异步加载**：List 内 10 个 Select，并发回想，不阻塞渲染

### 方案对比

| 方案 | 回调次数 | 实际请求次数 | 耦合度 | 复杂度 |
|------|---------|-------------|-------|-------|
| **旧版本** | 1 次（List 统一） | 1 次 | 高 | 高 |
| **新版本（无缓存）** | 10 次（各自独立） | 10 次 | 低 | 低 |
| **新版本（有缓存）** | 10 次（各自独立） | 2 次（去重后） | 低 | 低 |

---

## 实现方案

### 1. 回想缓存管理器

```typescript
/**
 * 回想缓存管理器
 * 职责：缓存 OnSelectFuzzy 回想结果，避免重复请求
 */
class RecallCacheManager {
  // 缓存存储：key = cacheKey, value = Promise<CallbackResult>
  private cache = new Map<string, Promise<CallbackResult>>()
  
  // 缓存超时时间（毫秒）
  private cacheTimeout = 5000  // 5 秒
  
  /**
   * 生成缓存 key
   * @param field 字段配置
   * @param value 回想值
   * @param context 上下文（可选）
   */
  private generateCacheKey(
    field: FieldConfig,
    value: any,
    context?: Record<string, any>
  ): string {
    // 🔥 缓存 key 包含：字段 code + 值 + 上下文哈希
    const contextHash = context ? this.hashObject(context) : ''
    
    // 处理值
    let valueKey: string
    if (Array.isArray(value)) {
      // MultiSelect: 排序后拼接
      valueKey = [...value].sort().join(',')
    } else {
      valueKey = String(value)
    }
    
    return `${field.code}:${valueKey}:${contextHash}`
  }
  
  /**
   * 对象哈希（简单实现）
   */
  private hashObject(obj: Record<string, any>): string {
    return JSON.stringify(obj)
      .split('')
      .reduce((hash, char) => {
        return ((hash << 5) - hash) + char.charCodeAt(0)
      }, 0)
      .toString(36)
  }
  
  /**
   * 获取缓存或执行回调
   * @param field 字段配置
   * @param value 回想值
   * @param context 表单上下文
   * @param executor 回调执行函数
   */
  async getOrFetch(
    field: FieldConfig,
    value: any,
    context: Record<string, any>,
    executor: () => Promise<CallbackResult>
  ): Promise<CallbackResult> {
    
    const cacheKey = this.generateCacheKey(field, value, context)
    
    // 🔥 检查缓存
    if (this.cache.has(cacheKey)) {
      console.log(`[RecallCache] 缓存命中: ${cacheKey}`)
      return this.cache.get(cacheKey)!
    }
    
    console.log(`[RecallCache] 缓存未命中，执行回调: ${cacheKey}`)
    
    // 🔥 执行回调（保存 Promise，支持并发请求复用）
    const promise = executor().catch(error => {
      // 失败时清除缓存
      this.cache.delete(cacheKey)
      throw error
    })
    
    this.cache.set(cacheKey, promise)
    
    // 🔥 设置缓存超时
    setTimeout(() => {
      this.cache.delete(cacheKey)
      console.log(`[RecallCache] 缓存过期: ${cacheKey}`)
    }, this.cacheTimeout)
    
    return promise
  }
  
  /**
   * 清空缓存
   */
  clear(): void {
    this.cache.clear()
    console.log('[RecallCache] 缓存已清空')
  }
}

// 全局单例
export const recallCacheManager = new RecallCacheManager()
```

### 2. SelectWidget 使用缓存回想

```typescript
class SelectWidget extends BaseWidget {
  /**
   * 根据值回想显示信息
   * 🔥 各自管各自，自动使用缓存
   */
  async recallByValue(value: any): Promise<void> {
    if (!value) return
    
    console.log(`[SelectWidget] ${this.fieldPath} 开始回想，值: ${value}`)
    
    try {
      // 🔥 通过缓存管理器执行回想
      const result = await recallCacheManager.getOrFetch(
        this.field,
        value,
        this.formManager.getAllValues(),  // 完整上下文
        async () => {
          // 实际回调执行函数
          return await callbackManager.executeSelectFuzzy(
            this.field,
            undefined,  // 不传 query
            this.formManager,
            'by_field_values'  // 🔥 标识：根据值回想
          )
        }
      )
      
      // 更新自己的值
      if (result.values.length > 0) {
        const option = result.values[0]
        
        this.onChange({
          raw: option.value,
          display: option.label,
          meta: {
            displayInfo: option.displayInfo,
            dataType: this.field.data.type,
            fromCallback: true
          }
        })
        
        console.log(`[SelectWidget] ${this.fieldPath} 回想成功`)
      } else {
        console.warn(`[SelectWidget] ${this.fieldPath} 回想失败：未找到值 ${value}`)
      }
      
    } catch (error) {
      console.error(`[SelectWidget] ${this.fieldPath} 回想失败:`, error)
    }
  }
  
  /**
   * 组件挂载时，检查是否需要回想
   */
  mounted() {
    const value = this.value.raw
    
    // 🔥 如果有值但没有 displayInfo，触发回想
    if (value && !this.value.meta?.displayInfo) {
      this.recallByValue(value)
    }
  }
}
```

### 3. MultiSelectWidget 使用缓存回想

```typescript
class MultiSelectWidget extends BaseWidget {
  /**
   * 根据值回想显示信息（批量）
   * 🔥 MultiSelect 可以一次性查询多个值（SQL IN）
   */
  async recallByValues(values: any[]): Promise<void> {
    if (!values || values.length === 0) return
    
    console.log(`[MultiSelectWidget] ${this.fieldPath} 开始回想，值: ${values}`)
    
    try {
      // 🔥 MultiSelect 的缓存 key 是所有值的组合
      const result = await recallCacheManager.getOrFetch(
        this.field,
        values,  // 数组值
        this.formManager.getAllValues(),
        async () => {
          // 实际回调执行函数
          return await callbackManager.executeSelectFuzzy(
            this.field,
            undefined,
            this.formManager,
            'by_field_values'  // 🔥 标识：根据值回想
          )
        }
      )
      
      // 更新自己的值
      if (result.values.length > 0) {
        this.onChange({
          raw: values,
          display: `已选 ${result.values.length} 项`,
          meta: {
            displayInfo: result.values.map(opt => opt.displayInfo),
            dataType: this.field.data.type,
            fromCallback: true
          }
        })
        
        console.log(`[MultiSelectWidget] ${this.fieldPath} 回想成功`)
      }
      
    } catch (error) {
      console.error(`[MultiSelectWidget] ${this.fieldPath} 回想失败:`, error)
    }
  }
  
  mounted() {
    const values = this.value.raw
    
    // 🔥 如果有值但没有 displayInfo，触发回想
    if (values && Array.isArray(values) && values.length > 0 && !this.value.meta?.displayInfo) {
      this.recallByValues(values)
    }
  }
}
```

### 4. CallbackManager 支持 by_field_values

```typescript
class CallbackManager {
  /**
   * 执行 OnSelectFuzzy 回调
   * @param inputType 输入类型：'fuzzy_search' | 'by_field_values'
   */
  async executeSelectFuzzy(
    field: FieldConfig,
    searchValue?: string,
    formManager?: ReactiveFormDataManager,
    inputType: 'fuzzy_search' | 'by_field_values' = 'fuzzy_search'
  ): Promise<CallbackResult> {
    
    const request = formManager ? formManager.prepareSubmitData() : {}
    
    // 🔥 区分输入类型
    const requestData: any = {
      code: field.code,
      _code: field.code,
      request: request,
      input_type: inputType,  // 🔥 标识回想类型
      value_type: this.getValueType(field.data.type)
    }
    
    if (inputType === 'fuzzy_search') {
      // 模糊搜索
      requestData.value = searchValue || ''
    } else {
      // 根据值回想
      // 🔥 value 从 request 中提取（后端会根据 code 自动获取）
      requestData.value = request[field.code]
    }
    
    console.log('[CallbackManager] OnSelectFuzzy 回调请求:', {
      field: field.code,
      inputType: inputType,
      value: requestData.value
    })
    
    const response = await post(
      `/api/v1/callback${router}?_type=OnSelectFuzzy&_method=${method}`,
      requestData
    )
    
    return {
      values: response.data.values || [],
      statistics: response.data.statistics,
      multiple: response.multiple || false
    }
  }
  
  /**
   * 获取值类型
   */
  private getValueType(dataType: string): string {
    if (dataType.includes('int')) return 'number'
    if (dataType.includes('float')) return 'number'
    if (dataType.includes('bool')) return 'boolean'
    return 'string'
  }
}
```

---

## 后端处理

### 后端识别 input_type

```go
// OnSelectFuzzy 回调处理
func OnSelectFuzzy(ctx *runner.Context, req *OnSelectFuzzyReq) (*OnSelectFuzzyResp, error) {
    // 🔥 根据 input_type 判断查询方式
    if req.InputType == "by_field_values" {
        // 回想模式：根据值查询
        return recallByValues(ctx, req)
    } else {
        // 模糊搜索模式
        return fuzzySearch(ctx, req)
    }
}

// 回想查询
func recallByValues(ctx *runner.Context, req *OnSelectFuzzyReq) (*OnSelectFuzzyResp, error) {
    // 从 request 中提取字段值
    fieldValue := req.Request[req.Code]
    
    var products []Product
    
    // 🔥 根据值类型生成 SQL
    if isArray(fieldValue) {
        // MultiSelect 或 List 批量回想
        values := fieldValue.([]interface{})
        // SQL: SELECT * FROM products WHERE product_id IN (1, 2, 3)
        db.Where("product_id IN ?", values).Find(&products)
    } else {
        // Select 单值回想
        value := fieldValue
        // SQL: SELECT * FROM products WHERE product_id = 1
        db.Where("product_id = ?", value).First(&products)
    }
    
    // 返回选项列表
    var options []*OnSelectFuzzyOption
    for _, product := range products {
        options = append(options, &OnSelectFuzzyOption{
            Value: product.ID,
            Label: product.Name,
            DisplayInfo: map[string]interface{}{
                "商品名称": product.Name,
                "价格":    product.Price,
                "库存":    product.Stock,
            },
        })
    }
    
    return &OnSelectFuzzyResp{
        Values: options,
        Statistics: map[string]string{
            "商品总价": "sum(价格,*quantity)",
        },
    }, nil
}
```

---

## 性能分析

### 场景 1：List 内 10 行，每行一个 Select

**数据分布：**
- 第 1-3 行：product_id = 1
- 第 4-6 行：product_id = 2
- 第 7 行：product_id = 3
- 第 8-10 行：product_id = 1

| 方案 | 回调次数 | 实际请求次数 | SQL 执行次数 | 总耗时 |
|------|---------|-------------|------------|-------|
| **旧版本（List 统一）** | 1 次 | 1 次 | 1 次（IN查询） | ~200ms |
| **新版本（无缓存）** | 10 次 | 10 次 | 10 次 | ~2000ms |
| **新版本（有缓存）** | 10 次 | **3 次**（去重后） | 3 次 | ~600ms |

**缓存命中率：**
```
总回调: 10 次
缓存命中: 7 次 (product_id=1 命中 6 次, product_id=2 命中 3 次)
实际请求: 3 次 (product_id=1, 2, 3 各一次)
命中率: 70%
```

### 场景 2：List 内 10 行，每行一个 MultiSelect

**数据分布：**
- 第 1 行：[1, 2, 3]
- 第 2 行：[1, 2, 3]  // 相同
- 第 3 行：[2, 3, 4]
- 第 4-10 行：[1, 2, 3]  // 相同

| 方案 | 回调次数 | 实际请求次数 | SQL 执行次数 | 总耗时 |
|------|---------|-------------|------------|-------|
| **旧版本（List 统一）** | 1 次 | 1 次 | 1 次（IN 1,2,3,4） | ~200ms |
| **新版本（无缓存）** | 10 次 | 10 次 | 10 次 | ~2000ms |
| **新版本（有缓存）** | 10 次 | **2 次**（去重后） | 2 次 | ~400ms |

**缓存命中率：**
```
总回调: 10 次
缓存命中: 8 次 ([1,2,3] 命中 8 次)
实际请求: 2 次 ([1,2,3] 和 [2,3,4] 各一次)
命中率: 80%
```

### 性能优化建议

1. **并发回想**：使用 `Promise.all()` 并发执行多个回想请求
2. **懒加载**：List 分页时，只回想当前页的数据
3. **缓存时间**：根据业务调整缓存超时时间（默认 5 秒）

---

## 完整流程示例

### 场景：用户 A 分享 URL 给用户 B

```typescript
// 1. 用户 A 的表单数据
{
  member_card_id: 1,
  products: [
    { product_id: 1, quantity: 2 },
    { product_id: 1, quantity: 1 },  // 重复
    { product_id: 4, quantity: 3 }
  ]
}

// 2. 生成 URL
const url = `https://example.com/form?member_card_id=1&products=[{"product_id":1,"quantity":2},{"product_id":1,"quantity":1},{"product_id":4,"quantity":3}]`

// 3. 用户 B 打开 URL
// FormRenderer.mounted() 被触发

// 4. FormRenderer 加载 URL 参数
formManager.loadFromUrlParams({
  member_card_id: 1,
  products: [
    { product_id: 1, quantity: 2 },
    { product_id: 1, quantity: 1 },
    { product_id: 4, quantity: 3 }
  ]
})

// 5. SelectWidget 检测到需要回想
// member_card_id SelectWidget
selectWidget1.mounted() {
  // value = 1, 但没有 displayInfo
  this.recallByValue(1)
  // → 缓存 key: "member_card_id:1:xxxx"
  // → 实际请求: 1 次
  // → 结果: { value: 1, label: "金卡会员", displayInfo: {...} }
}

// 6. List 内 SelectWidget 并发回想
// products[0].product_id
selectWidget2.mounted() {
  this.recallByValue(1)
  // → 缓存 key: "product_id:1:xxxx"
  // → 实际请求: 1 次 ✅ 第一次请求
}

// products[1].product_id
selectWidget3.mounted() {
  this.recallByValue(1)
  // → 缓存 key: "product_id:1:xxxx"
  // → 🔥 缓存命中！不发送请求
}

// products[2].product_id
selectWidget4.mounted() {
  this.recallByValue(4)
  // → 缓存 key: "product_id:4:xxxx"
  // → 实际请求: 1 次 ✅ 第二次请求
}

// 7. 总结
// 回调次数: 4 次（1 个会员卡 + 3 个商品）
// 实际请求: 3 次（会员卡 1 次，商品 2 次）
// 缓存命中: 1 次（product_id=1 第二次命中缓存）
// 性能提升: 25%
```

---

## 缓存策略对比

| 策略 | 优点 | 缺点 | 适用场景 |
|------|------|------|---------|
| **无缓存** | 实现简单 | 重复请求多，性能差 | ❌ 不推荐 |
| **组件级缓存** | 解耦，各自管各自 | 需要缓存管理器 | ✅ **推荐**（新方案） |
| **List 统一处理** | 请求少 | 耦合度高，复杂 | ❌ 不推荐（旧方案） |

---

## 实现优先级

### Phase 1：基础回想（必须）
- ✅ SelectWidget 支持 `recallByValue()`
- ✅ MultiSelectWidget 支持 `recallByValues()`
- ✅ CallbackManager 支持 `input_type: "by_field_values"`
- ✅ 后端识别 `input_type` 并执行相应查询

### Phase 2：缓存层（推荐）
- ✅ `RecallCacheManager` 实现
- ✅ 自动缓存复用
- ✅ 缓存超时机制
- ✅ 并发请求合并

### Phase 3：性能优化（可选）
- ⭐ 懒加载（分页场景）
- ⭐ 缓存持久化（LocalStorage）

---

## 总结

### 核心优势

| 特性 | 说明 | 优势 |
|------|------|------|
| **解耦** | 各自管各自，ListWidget 不操作子组件 | ⭐⭐⭐⭐⭐ |
| **简单** | 每个组件独立回想，逻辑清晰 | ⭐⭐⭐⭐⭐ |
| **性能** | 缓存复用，避免重复请求 | ⭐⭐⭐⭐ |
| **扩展性** | 新增组件类型无需修改 ListWidget | ⭐⭐⭐⭐⭐ |
| **可维护性** | 代码集中，易于调试 | ⭐⭐⭐⭐⭐ |

### 技术与维护性的平衡

✅ **技术先进**：缓存策略、并发优化  
✅ **维护简单**：解耦设计、职责清晰  
✅ **性能优秀**：缓存命中率高（70-80%）  
✅ **扩展方便**：新增组件无需修改现有代码  

### 最终建议

**推荐方案：各自管各自 + 缓存层**

- List 内 10 行 → 回想 10 次
- 缓存自动去重 → 实际请求 2-3 次
- MultiSelect 批量查询 → SQL IN 一次搞定
- 性能损失：~200-400ms（完全可接受）
- 架构收益：解耦、简单、易维护

**这是技术与维护性的最佳平衡点！** 🎉
