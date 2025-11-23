# Slider 组件设计分析

## 一、组件概述

Slider（滑块/进度条）组件是一个多功能组件，根据使用场景自动切换显示模式：
- **输入模式**：显示为滑块（slider bar），用于编辑/新增表单
- **输出模式**：显示为进度条（progress bar），用于表格列表、详情展示

## 二、组件参数设计

### 2.1 基础参数（必需）

| 参数 | 类型 | 说明 | 示例 |
|------|------|------|------|
| `min` | float64 | 最小值（必需） | `0` |
| `max` | float64 | 最大值（必需） | `100` |

### 2.2 基础参数（可选）

| 参数 | 类型 | 默认值 | 说明 | 示例 |
|------|------|--------|------|------|
| `step` | float64 | `1` | 步长（拖动时的步进值） | `0.1`, `5`, `10` |
| `default` | float64 | - | 默认值 | `50` |
| `unit` | string | `""` | 单位（显示在值后面） | `%`, `元`, `kg`, `分` |

### 2.3 显示参数（可选）

| 参数 | 类型 | 默认值 | 说明 | 示例 |
|------|------|--------|------|------|
| `show_input` | bool | `false` | 是否显示输入框（滑块旁边） | `true` |
| `show_stops` | bool | `false` | 是否显示刻度标记 | `true` |
| `show_tooltip` | bool | `true` | 是否显示提示（拖动时） | `false` |
| `format_tooltip` | string | `""` | 自定义提示格式（支持 `{value}` 占位符） | `{value}%`, `{value}分` |

### 2.4 输出模式参数（进度条，可选）

| 参数 | 类型 | 默认值 | 说明 | 示例 |
|------|------|--------|------|------|
| `show_percentage` | bool | `true` | 是否显示百分比 | `false` |
| `status` | string | `""` | 状态颜色（success/warning/danger/info） | `success`, `warning` |
| `stroke_width` | int | `6` | 进度条粗细（像素） | `8`, `10` |

## 三、使用场景分析

### 3.1 输入模式（编辑/新增表单）

**显示方式**：滑块（slider bar）

**使用示例**：
```go
// 进度字段：0-100，步长1，显示输入框
Progress int `json:"progress" gorm:"column:progress" widget:"name:完成进度;type:slider;min:0;max:100;step:1;show_input:true;unit:%" validate:"min=0,max=100"`

// 评分字段：0-10，步长0.5，显示刻度
Score float64 `json:"score" gorm:"column:score" widget:"name:评分;type:slider;min:0;max:10;step:0.5;show_stops:true;unit:分" validate:"min=0,max=10"`

// 温度字段：-20到40，步长1，自定义提示
Temperature float64 `json:"temperature" gorm:"column:temperature" widget:"name:温度;type:slider;min:-20;max:40;step:1;format_tooltip:{value}°C;unit:°C"`
```

### 3.2 输出模式（表格列表/详情）

**显示方式**：进度条（progress bar）

**使用示例**：
```go
// 进度字段：显示为进度条，带百分比，成功状态
Progress int `json:"progress" gorm:"column:progress" widget:"name:完成进度;type:slider;min:0;max:100;show_percentage:true;status:success"`

// 评分字段：显示为进度条，不带百分比，警告状态
Score float64 `json:"score" gorm:"column:score" widget:"name:评分;type:slider;min:0;max:10;show_percentage:false;status:warning"`
```

### 3.3 搜索模式

**显示方式**：范围输入框（两个输入框：最小值、最大值）

**搜索类型**：支持 `gte`（大于等于）和 `lte`（小于等于）

**使用示例**：
```go
// 进度搜索：支持范围搜索
Progress int `json:"progress" gorm:"column:progress" widget:"name:完成进度;type:slider;min:0;max:100" search:"gte,lte"`
```

**前端实现**：
- 使用两个 `el-input-number` 或 `el-slider` 的 range 模式
- 最小值对应 `gte` 搜索
- 最大值对应 `lte` 搜索
- URL 参数格式：`gte=progress:50&lte=progress:80`（进度在50-80之间）

## 四、前端实现建议

### 4.1 输入模式（SliderWidget.vue）

```vue
<template>
  <div class="slider-widget">
    <!-- 编辑模式：滑块 -->
    <el-slider
      v-if="mode === 'edit'"
      v-model="internalValue"
      :min="min"
      :max="max"
      :step="step"
      :show-input="showInput"
      :show-stops="showStops"
      :show-tooltip="showTooltip"
      :format-tooltip="formatTooltipFunc"
      :disabled="field.widget?.config?.disabled"
      @change="handleChange"
    />
    
    <!-- 输出模式：进度条 -->
    <el-progress
      v-else-if="mode === 'table-cell' || mode === 'detail'"
      :percentage="percentage"
      :status="status"
      :stroke-width="strokeWidth"
      :show-text="showPercentage"
    />
    
    <!-- 搜索模式：范围输入 -->
    <div v-else-if="mode === 'search'" class="slider-search">
      <el-input-number
        v-model="minValue"
        :min="min"
        :max="max"
        :step="step"
        :placeholder="`最小${field.name}`"
        @change="handleSearchChange"
      />
      <span class="separator">-</span>
      <el-input-number
        v-model="maxValue"
        :min="min"
        :max="max"
        :step="step"
        :placeholder="`最大${field.name}`"
        @change="handleSearchChange"
      />
    </div>
  </div>
</template>
```

### 4.2 关键计算属性

```typescript
// 计算百分比（用于进度条显示）
const percentage = computed(() => {
  const value = props.value?.raw ?? 0
  const range = max - min
  if (range === 0) return 0
  return Math.round(((value - min) / range) * 100)
})

// 格式化提示
const formatTooltipFunc = computed(() => {
  if (formatTooltip) {
    return (value: number) => {
      return formatTooltip.replace('{value}', String(value))
    }
  }
  return undefined
})
```

## 五、数据类型建议

**推荐数据类型**：
- `int`：整数类型（如：进度、评分）
- `float`：浮点数类型（如：温度、价格）

**不推荐**：
- `string`：字符串类型不适合滑块

## 六、验证规则建议

```go
// 进度字段：0-100
Progress int `json:"progress" widget:"type:slider;min:0;max:100" validate:"min=0,max=100"`

// 评分字段：0-10，步长0.5
Score float64 `json:"score" widget:"type:slider;min:0;max:10;step:0.5" validate:"min=0,max=10"`
```

## 七、总结

Slider 组件是一个灵活的组件，通过不同的参数配置可以适应多种场景：
1. **输入场景**：滑块输入，适合需要快速调整数值的场景
2. **输出场景**：进度条展示，直观显示完成度、评分等
3. **搜索场景**：范围搜索，支持区间查询

**核心优势**：
- 用户体验好：滑块操作直观，进度条展示清晰
- 功能完整：支持输入、输出、搜索三种模式
- 配置灵活：丰富的参数满足不同需求

