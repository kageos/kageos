# Statistics 聚合统计函数包

> **功能**：提供类型安全的聚合统计表达式构建函数，用于 `OnSelectFuzzyResp.Statistics`

---

## 📋 快速开始

### 导入包

```go
import "github.com/ai-agent-os/ai-agent-os/sdk/agent-app/statistics"
```

### 使用方式：直接函数调用

```go
Statistics: map[string]interface{}{
    "商品原价总额(元)":  statistics.Sum("价格,*quantity"),
    "会员折扣后价格(元)": statistics.Sum("价格,*quantity,*0.9"),
    "商品种类数":      statistics.Count("价格"),
    "商品总数量(件)":   statistics.Sum("quantity"),
    "会员折扣":       "9折优惠",  // 静态信息，直接写字符串
}
```

**关键点**：
- ✅ 操作符必须明确指定（如 `*quantity`、`*0.9`），这样更直观
- ✅ 便于后续支持 `+`、`-`、`/` 等操作符
- ✅ 表达式格式：`主字段,操作符字段1,操作符字段2,...`

---

## 📚 函数列表

### 基础聚合函数

| 函数 | 说明 | 示例 |
|------|------|------|
| `Sum(expression)` | 求和 | `Sum("价格,*quantity")` → `"sum(价格,*quantity)"` |
| `Count(field)` | 计数 | `Count("价格")` → `"count(价格)"` |
| `Avg(expression)` | 平均值 | `Avg("价格,*quantity")` → `"avg(价格,*quantity)"` |
| `Min(field)` | 最小值 | `Min("价格")` → `"min(价格)"` |
| `Max(field)` | 最大值 | `Max("价格")` → `"max(价格)"` |

**表达式格式说明**：
- `"价格"` - 单个字段
- `"价格,*quantity"` - 价格 × 数量
- `"价格,*quantity,*0.9"` - 价格 × 数量 × 0.9
- 操作符必须明确指定（`*` 表示乘法），便于后续支持 `+`、`-`、`/` 等

### List 层聚合函数

| 函数 | 说明 | 示例 |
|------|------|------|
| `ListSum(field)` | List 层求和 | `ListSum("用户总价")` → `"list_sum(用户总价)"` |
| `ListAvg(field)` | List 层平均值 | `ListAvg("用户总价")` → `"list_avg(用户总价)"` |
| `ListCount()` | List 层计数 | `ListCount()` → `"list_count()"` |

### 选中项字段值函数

| 函数 | 说明 | 示例 |
|------|------|------|
| `Value(field)` | 显示选中项的字段值（动态值） | `Value("余额")` → `"value(余额)"` |

**使用说明**：
- 用于单选场景，显示当前选中项的某个字段值
- 字段名来自 `DisplayInfo` 的 key（如 `"余额"`、`"折扣"`）
- 前端会从选中项的 `DisplayInfo` 中获取对应的值并显示
- 多选场景会显示第一个选中项的值
- 返回值可以是任何类型（文本、数字、日期等）

---

## 🎯 使用场景

### 场景1：基础求和

```go
// 计算所有商品的价格总和
statistics.Sum("价格")

// 计算所有商品的 价格×数量 总和
statistics.Sum("价格,*quantity")

// 计算所有商品的 价格×数量×0.9 总和（9折）
statistics.Sum("价格,*quantity,*0.9")
```

### 场景2：条件表达式（与 MySQL/SQL 一致）

表达式语法与 MySQL/SQL 保持一致，避免大模型混淆。条件用 **MySQL IF(cond, thenExpr, elseExpr)**：

```go
// 有输入价用 输入价×数量，否则用默认销售价×数量
statistics.Sum("IF(price > 0, price * quantity, 销售价 * quantity)")
```

### 场景3：计数

```go
// 计算选中了几种不同的商品
statistics.Count("价格")
```

### 场景4：平均值

```go
// 计算平均价格
statistics.Avg("价格")

// 计算平均总价（价格×数量）
statistics.Avg("价格,*quantity")
```

### 场景5：最值

```go
// 最低价格
statistics.Min("价格")

// 最高价格
statistics.Max("价格")
```

### 场景6：List 层聚合（MultiSelect 二层聚合）

```go
// 对所有行的"用户总价"字段求和
statistics.ListSum("用户总价")

// 对所有行的"用户总价"字段求平均值
statistics.ListAvg("用户总价")

// 计算有多少行
statistics.ListCount()
```

### 场景7：显示选中项的字段值（动态值）

```go
// 显示当前选中会员卡的"余额"
statistics.Value("余额")

// 显示当前选中会员卡的"折扣"
statistics.Value("折扣")

// 显示当前选中会员卡的"有效期"
statistics.Value("有效期")
```

---

## 📝 完整示例

### 示例1：收银台商品统计

```go
import "github.com/ai-agent-os/ai-agent-os/sdk/agent-app/statistics"

func onSelectFuzzyProduct(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
    // ... 查询商品逻辑 ...
    
    return &callback.OnSelectFuzzyResp{
        MaxSelections: 0,
        Items:         items,
        Statistics: map[string]interface{}{
            "商品原价总额(元)":  statistics.Sum("价格,*quantity"),
            "会员折扣后价格(元)": statistics.Sum("价格,*quantity,*0.9"),
            "优惠金额(元)":    statistics.Sum("价格,*quantity,*0.1"),
            "商品种类数":      statistics.Count("价格"),
            "商品总数量(件)":   statistics.Sum("quantity"),
            "会员折扣":       "9折优惠",
            "配送说明":       "满99元包邮，不满99元运费10元",
        },
    }, nil
}
```

### 示例2：会员卡选择（动态显示余额）

```go
import "github.com/ai-agent-os/ai-agent-os/sdk/agent-app/statistics"

func onSelectFuzzyMemberCard(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
    // ... 查询会员卡逻辑 ...
    // 每个会员卡的 DisplayInfo 包含：余额、折扣、有效期等
    
    return &callback.OnSelectFuzzyResp{
        MaxSelections: 1,  // 单选
        Items:         items,
        Statistics: map[string]interface{}{
            "当前余额":    statistics.Value("余额"),     // 动态显示选中会员卡的余额
            "当前折扣":    statistics.Value("折扣"),     // 动态显示选中会员卡的折扣
            "有效期至":    statistics.Value("有效期"),   // 动态显示选中会员卡的有效期
            "使用说明":    "余额不足时自动充值",
        },
    }, nil
}
```


---

## 🔍 查找可用函数

### 方式1：使用 go doc

```bash
go doc github.com/ai-agent-os/ai-agent-os/sdk/agent-app/statistics
```

### 方式2：IDE 自动补全

在代码中输入 `statistics.`，IDE 会自动提示所有可用函数。

### 方式3：查看源码

查看 `sdk/agent-app/statistics/statistics.go` 文件，所有函数都有详细注释。

---

## ⚠️ 注意事项

### 1. 字段名来源

- **基础字段**：来自 `DisplayInfo` 的 key（如 `"价格"`、`"库存"`）
- **行内字段**：来自同一行的字段名（如 `"quantity"`）
- **List 层字段**：来自行的计算结果字段名（如 `"用户总价"`）

### 2. 操作符

- **乘法**：使用 `*` 前缀（如 `*quantity`、`*0.9`）
- **必须明确指定**：操作符必须在表达式中明确写出，这样更直观
- **便于扩展**：后续可以支持 `+`、`-`、`/` 等操作符

### 3. 静态信息

- 非表达式的静态文本（如 `"9折优惠"`）可以直接写字符串

### 4. 表达式格式

- 函数生成的表达式格式与前端 `ExpressionParser` 完全兼容
- 不需要担心格式问题

---

## 📖 参考文档

- **前端实现**：`web/src/core/utils/ExpressionParser.ts`
- **使用示例**：`namespace/luobei/demo/code/api/tools/tools_cashier.go`
- **代码生成指南**：`blueprint/05-代码生成快速指南.md`

---

**最后更新**：2025-01-XX

