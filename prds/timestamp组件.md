我们的timestamp 组件涵盖了整个时间日期的选择，注意整个时间日期的值都是时间戳，且是毫秒级别的时间戳，整个系统只有这一个时间相关的组件
这个timestamp涵盖了所有
包括 时间选择，日期选择，年份选择，月份选择，时间日期，年份日期等等所有形式，只是通过format来进行格式化
分为两种时间类型
一种是绝对时间：例如
YYYY-MM-DD HH:mm:ss 这种是精确的时间，需要传递精确的时间戳

HH:mm:ss 这种是相对时间，不关心具体的年份月日等等，这种传递时间戳，需要传递 1970 1月1日的 HH:mm:ss 的时间，
为啥要这样干？因为这样后端才能根据这个时间进行排序，假如这个字段存储的是下课时间，那么我们需要根据下课时间进行排序的话就非常方便了
如果用绝对时间戳的方式存储的话，后面新增的记录即使下课早也会被排在后面，这是一个点，
所以这里先分清楚哪些是绝对时间，哪些是相对时间




因为有 table_permission: "read" 说明这个字段只在 table 列表中展示，在新增和更新表单中都不显示（因为是自动生成或只读字段）
```json

{
                "callbacks": null,
                "children": null,
                "code": "created_at",
                "data": {
                    "example": "",
                    "format": "",
                    "type": "int"
                },
                "desc": "",
                "name": "创建时间",
                "search": "",
                "table_permission": "read",
                "validation": "",
                "widget": {
                    "config": {
                        "disabled": false, //这个字段标识是否能进行时间的选择
                        "format": "YYYY-MM-DD HH:mm:ss"
                    },
                    "type": "timestamp"
                }
            }
```

这个字段无论是在table的新增修改还是在form函数的表单中都是可以调出时间日期选择器来选择时间的，
```json

{
                "callbacks": null,
                "children": null,
                "code": "push_at",
                "data": {
                    "example": "",
                    "format": "",
                    "type": "int"
                },
                "desc": "",
                "name": "发布时间",
                "search": "",
                "table_permission": "",
                "validation": "",
                "widget": {
                    "config": {
                        "disabled": false,
                        "format": "YYYY-MM-DD HH:mm:ss"
                    },
                    "type": "timestamp"
                }
            }
```

---

## table_permission 字段说明

> **详细说明请参考**：[table_permission权限逻辑.md](./table_permission权限逻辑.md)

### 快速参考

`table_permission` 控制字段在不同场景下的显示和编辑权限：

| 值 | Table 列表 | Table 新增 | Table 编辑 | Form 函数 |
|---|-----------|----------|----------|---------|
| `""` (空) | ✅ 显示 | ✅ 可填写 | ✅ 可修改 | ✅ 显示 |
| `"read"` | ✅ 显示 | ❌ 不显示 | ❌ 不显示 | ❌ 不显示 |
| `"create"` | ✅ 显示 | ✅ 可填写 | ❌ 不显示 | ✅ 显示 |
| `"update"` | ✅ 显示 | ❌ 不显示 | ✅ 可修改 | ❌ 不显示 |

### 典型时间字段场景

#### 1. `created_at` - 创建时间（只读）
```json
{
  "code": "created_at",
  "name": "创建时间",
  "table_permission": "read",  // 🔥 只读字段
  "widget": {
    "type": "timestamp",
    "config": {
      "format": "YYYY-MM-DD HH:mm:ss"
    }
  }
}
```

**业务逻辑**：后端自动生成，用户不可填写和修改

#### 2. `push_at` - 发布时间（可编辑）
```json
{
  "code": "push_at",
  "name": "发布时间",
  "table_permission": "",  // 🔥 全部权限
  "widget": {
    "type": "timestamp",
    "config": {
      "format": "YYYY-MM-DD HH:mm:ss"
    }
  }
}
```

**业务逻辑**：用户可以在新增和编辑时选择发布时间

---

## 前端实现技术方案

### 1. 核心设计理念

```
统一的时间组件 (TimestampWidget)
├─ 唯一的数据格式：毫秒级时间戳（int64）
├─ 多样的显示格式：通过 format 控制
└─ 两种时间语义：绝对时间 vs 相对时间
```

### 2. 两种时间类型对比

| 类型 | Format 示例 | 存储值示例 | 应用场景 |
|------|------------|-----------|---------|
| **绝对时间** | `YYYY-MM-DD HH:mm:ss` | `1698825600000`（2023-11-01 16:00:00） | 创建时间、发布时间、订单时间 |
| **相对时间** | `HH:mm:ss` | `46800000`（1970-01-01 13:00:00） | 上课时间、下课时间、营业时间 |

### 3. Format 到选择器的映射

| Format | Element Plus 组件类型 | 显示效果 | 时间类型 |
|--------|---------------------|---------|---------|
| `YYYY-MM-DD HH:mm:ss` | `datetime` | `2024-11-02 13:30:45` | 绝对 |
| `YYYY-MM-DD` | `date` | `2024-11-02` | 绝对 |
| `HH:mm:ss` | `time` | `13:30:45` | 相对 |
| `HH:mm` | `time` | `13:30` | 相对 |
| `YYYY-MM` | `month` | `2024-11` | 绝对 |
| `YYYY` | `year` | `2024` | 绝对 |

### 4. 前端实现伪代码

```typescript
class TimestampWidget extends BaseWidget {
  // 🔥 判断选择器类型
  private getPickerType(): 'datetime' | 'date' | 'time' | 'month' | 'year' {
    const format = this.config.format || 'YYYY-MM-DD HH:mm:ss'
    
    if (format.includes('HH') || format.includes('mm') || format.includes('ss')) {
      if (format.includes('YYYY') || format.includes('MM') || format.includes('DD')) {
        return 'datetime'  // 日期时间选择器
      }
      return 'time'  // 纯时间选择器（相对时间）
    }
    if (format === 'YYYY-MM') return 'month'
    if (format === 'YYYY') return 'year'
    return 'date'
  }

  // 🔥 判断是否为相对时间
  private isRelativeTime(): boolean {
    const format = this.config.format || 'YYYY-MM-DD HH:mm:ss'
    // 只有时分秒，没有年月日 → 相对时间
    return !(format.includes('YYYY') || format.includes('MM') || format.includes('DD'))
  }

  // 🔥 时间戳 → Date 对象（用于显示）
  private timestampToDate(timestamp: number): Date {
    return new Date(timestamp)  // 毫秒级时间戳
  }

  // 🔥 Date 对象 → 时间戳（用于提交）
  private dateToTimestamp(date: Date): number {
    if (this.isRelativeTime()) {
      // 相对时间：只取时分秒，日期设为 1970-01-01
      const hours = date.getHours()
      const minutes = date.getMinutes()
      const seconds = date.getSeconds()
      return (hours * 3600 + minutes * 60 + seconds) * 1000
    }
    
    // 绝对时间：完整时间戳
    return date.getTime()
  }

  // 🔥 提交时转换
  getRawValueForSubmit(): number | null {
    const currentValue = this.getValue()
    if (!currentValue?.raw) return null
    
    return typeof currentValue.raw === 'number' 
      ? currentValue.raw 
      : this.dateToTimestamp(new Date(currentValue.raw))
  }

  // 🔥 渲染
  render() {
    const pickerType = this.getPickerType()
    const currentValue = this.getValue()
    const dateValue = currentValue?.raw 
      ? this.timestampToDate(currentValue.raw) 
      : null

    return h(ElDatePicker, {
      type: pickerType,
      modelValue: dateValue,
      format: this.config.format || 'YYYY-MM-DD HH:mm:ss',
      valueFormat: 'x',  // 返回时间戳（毫秒）
      disabled: this.config.disabled,
      placeholder: `请选择${this.field.name}`,
      onChange: (value: number | null) => {
        this.updateRawValue(value)
      }
    })
  }
}
```

### 5. 相对时间的核心逻辑

**为什么相对时间要存储为 1970-01-01 的时间戳？**

```typescript
// 场景：课程表的下课时间

// ❌ 错误方式：存储字符串 "13:00:00"
// 问题：
// 1. 无法排序（字符串比较不准确）
// 2. 无法进行时间计算
// 3. 数据库索引效率低

// ✅ 正确方式：存储为 1970-01-01 13:00:00 的时间戳
const relativeTime = new Date('1970-01-01T13:00:00Z').getTime()  // 46800000

// 优势：
// 1. 可以直接排序（数字比较）
// 2. 不受记录创建日期影响
// 3. 跨日期可比较
// 4. 数据库原生支持时间类型

// 实际案例：
记录1：2024-10-01 创建，下课时间 13:00 → 存储 46800000
记录2：2025-11-01 创建，下课时间 12:30 → 存储 45000000

// 排序查询：SELECT * FROM courses ORDER BY end_time
// 结果：记录2 (12:30) 自动排在 记录1 (13:00) 前面 ✅
```

### 6. 实现难点和解决方案

#### 难点 1：相对时间的 UI 展示

**问题**：ElDatePicker 的 `type="time"` 可能会显示完整日期 `1970-01-01 13:00:00`，用户会困惑。

**解决方案**：
```typescript
// 方案 1：使用 ElTimePicker（推荐）
if (this.isRelativeTime()) {
  return h(ElTimePicker, {
    modelValue: currentValue?.raw,
    format: this.config.format,  // HH:mm:ss
    onChange: (value: Date) => {
      const timestamp = this.dateToTimestamp(value)
      this.updateRawValue(timestamp)
    }
  })
}

// 方案 2：自定义 format 隐藏日期部分
// Element Plus 会自动根据 format 只显示时间部分
```

#### 难点 2：初始化值的类型兼容

**问题**：后端可能返回数字或字符串。

**解决方案**：
```typescript
initValue(backendValue: any) {
  if (!backendValue) {
    this.updateRawValue(null)
    return
  }
  
  const timestamp = typeof backendValue === 'string' 
    ? parseInt(backendValue, 10) 
    : backendValue
    
  this.updateRawValue(timestamp)
}
```

#### 难点 3：类型转换确保返回数字

**问题**：提交时可能是 Date 对象或字符串。

**解决方案**：
```typescript
protected convertValueByType(value: any): any {
  if (value === null || value === undefined || value === '') {
    return null
  }
  
  // Date 对象 → 时间戳
  if (value instanceof Date) {
    return this.dateToTimestamp(value)
  }
  
  // 字符串/数字 → 数字
  const timestamp = typeof value === 'string' 
    ? parseInt(value, 10) 
    : value
    
  return isNaN(timestamp) ? null : timestamp
}
```

### 7. 测试场景

#### 绝对时间测试
```json
// 输入：2024-11-02 13:30:45
// 存储：1698912645000
// 显示：2024-11-02 13:30:45 ✅
```

#### 相对时间测试
```json
// 输入：13:30:45
// 存储：48645000（1970-01-01 13:30:45 的时间戳）
// 显示：13:30:45 ✅

// 排序测试：
// 12:30:00 (45000000) < 13:30:45 (48645000) < 14:00:00 (50400000) ✅
```

#### 边界情况测试
```typescript
// null/undefined → null ✅
// 空字符串 → null ✅
// 无效时间戳 → null ✅
// disabled=true → 只读显示 ✅
```

### 8. 复杂度评估

| 功能点 | 难度 | 工作量 | 优先级 |
|--------|------|--------|--------|
| **基础渲染** | ⭐⭐ | 30 分钟 | P0 |
| **Format 解析** | ⭐⭐ | 20 分钟 | P0 |
| **绝对时间处理** | ⭐ | 10 分钟 | P0 |
| **相对时间处理** | ⭐⭐⭐ | 1 小时 | P1 |
| **Disabled 状态** | ⭐ | 5 分钟 | P0 |
| **类型转换** | ⭐⭐ | 20 分钟 | P0 |
| **总计** | ⭐⭐⭐ | **2-2.5 小时** | - |

### 9. 实现建议

1. **先实现绝对时间**（80% 场景）
   - `YYYY-MM-DD HH:mm:ss`
   - `YYYY-MM-DD`
   - `YYYY-MM`
   - `YYYY`

2. **再实现相对时间**（20% 场景）
   - `HH:mm:ss`
   - `HH:mm`

3. **充分测试边界情况**
   - null 值处理
   - 空字符串处理
   - 无效时间戳处理
   - 不同 format 组合

4. **性能优化**
   - 缓存 format 解析结果
   - 避免重复的时间戳转换

### 10. 注册到 WidgetFactory

```typescript
// web/src/core/factories/WidgetFactory.ts
this.registerWidget('timestamp', TimestampWidget)
```

---

## 总结

**Timestamp 组件的设计亮点**：
1. ✅ **统一性**：唯一的时间组件，降低学习成本
2. ✅ **灵活性**：通过 `format` 支持所有时间类型
3. ✅ **数据一致性**：统一用时间戳存储，便于比较和排序
4. ✅ **相对时间的巧妙设计**：1970 基准日期，完美解决纯时间排序问题
5. ✅ **扩展性**：符合现有 Widget 架构，实现成本低