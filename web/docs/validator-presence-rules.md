# Validator Presence Rules

## 目标

前端将 `go-playground/validator/v10` 中与“字段是否允许出现、是否当前必填”相关的规则，统一映射为三类运行时状态：

- `visible`: 字段当前是否展示
- `required`: 字段当前是否展示为必填
- `excluded`: 字段当前是否禁止出现

这套语义同时作用于：

- 顶层表单显示
- 嵌套 `form` 显示
- `table` 行内字段显示
- 提交前前端校验
- 提交 payload 清理

## 当前支持的规则

- `required`
- `required_if`
- `required_unless`
- `required_with`
- `required_with_all`
- `required_without`
- `required_without_all`
- `excluded_if`
- `excluded_unless`
- `excluded_with`
- `excluded_with_all`
- `excluded_without`
- `excluded_without_all`

## 行为定义

### `required`

- 始终显示
- 始终为必填

示例：

```go
Receiver string `validate:"required"`
```

### `required_if`

- 当所有条件字段都等于指定值时，字段显示并标记为必填
- 否则隐藏，但不主动清值

示例：

```go
TaxNo string `validate:"required_if=InvoiceType company"`
```

### `required_unless`

- 除非所有条件字段都等于指定值，否则字段显示并标记为必填

示例：

```go
Address string `validate:"required_unless=DeliveryType pickup"`
```

### `required_with`

- 当任一引用字段有值时，字段显示并标记为必填

示例：

```go
ContactName string `validate:"required_with=Phone Email"`
```

### `required_with_all`

- 当所有引用字段都有值时，字段显示并标记为必填

示例：

```go
TaxNo string `validate:"required_with_all=CompanyName CompanyBank"`
```

### `required_without`

- 当任一引用字段为空时，字段显示并标记为必填

示例：

```go
Email string `validate:"required_without=Phone"`
```

### `required_without_all`

- 当所有引用字段都为空时，字段显示并标记为必填

示例：

```go
ContactChannel string `validate:"required_without_all=Phone Email"`
```

### `excluded_if`

- 当所有条件字段都等于指定值时，字段隐藏并清空
- 提交 payload 中会剔除该字段

示例：

```go
PickupStoreID string `validate:"excluded_if=DeliveryType home"`
```

### `excluded_unless`

- 除非所有条件字段都等于指定值，否则字段隐藏并清空

示例：

```go
TaxNo string `validate:"excluded_unless=InvoiceType company"`
```

### `excluded_with`

- 当任一引用字段有值时，字段隐藏并清空

示例：

```go
SmsCode string `validate:"excluded_with=Password"`
```

### `excluded_with_all`

- 当所有引用字段都有值时，字段隐藏并清空

示例：

```go
ManualPrice string `validate:"excluded_with_all=SkuID PromotionID"`
```

### `excluded_without`

- 当任一引用字段为空时，字段隐藏并清空

示例：

```go
InviteCode string `validate:"excluded_without=CampaignID"`
```

### `excluded_without_all`

- 当所有引用字段都为空时，字段隐藏并清空

示例：

```go
GeoPoint string `validate:"excluded_without_all=Province City"`
```

## 约束

- 条件字段引用优先使用 Go 字段名，通过 `field_name` 映射到前端 `code`
- `required_if / required_unless / excluded_if / excluded_unless` 必须是成对参数：`Field value`
- `required_with* / required_without* / excluded_with* / excluded_without*` 使用空格分隔字段列表
- 暂不建议条件值使用带空格的展示文案，推荐使用稳定枚举值
- `excluded_*` 优先级高于 `required_*`

## 空值语义

前端存在性判断按接近 `validator/v10` 的零值语义处理：

- `null` / `undefined` 视为空
- `""` 视为空
- `0` 视为空
- `false` 视为空
- 空数组、空对象视为空

## 当前不做的事

- 不把 `min/max/email/oneof/eqfield` 这类规则转成显隐条件
- 不支持自定义 validator 参与显隐
- 不处理带空格条件值的复杂转义语法
