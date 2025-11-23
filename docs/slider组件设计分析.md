# Slider 组件设计分析

## 一、组件概述

Slider（滑块/进度条）组件是一个多功能组件，根据使用场景自动切换显示模式：
- **输入模式**：显示为滑块（slider bar），用于编辑/新增表单
- **输出模式**：显示为进度条（progress bar），用于表格列表、详情展示
- **搜索模式**：自动支持范围搜索（gte/lte）

**设计理念**：参数尽可能少，必要的功能默认开启，降低学习成本。

## 二、组件参数设计（极简版）

### 2.1 必需参数

| 参数 | 类型 | 说明 | 示例 |
|------|------|------|------|
| `min` | float64 | 最小值（必需） | `0` |
| `max` | float64 | 最大值（必需） | `100` |

### 2.2 可选参数（有合理默认值）

| 参数 | 类型 | 默认值 | 说明 | 示例 |
|------|------|--------|------|------|
| `step` | float64 | `1` | 步长（拖动时的步进值） | `0.1`, `5`, `10` |
| `default` | float64 | - | 默认值 | `50` |
| `unit` | string | `""` | 单位（显示在值后面） | `%`, `元`, `kg`, `分` |

### 2.3 默认行为（前端自动处理，无需配置）

| 功能 | 默认值 | 说明 |
|------|--------|------|
| 显示输入框 | `false` | 简单场景不需要，保持界面简洁 |
| 显示刻度 | `false` | 简单场景不需要，保持界面简洁 |
| 显示提示 | `true` | 拖动时自动显示当前值（带单位） |
| 显示百分比 | `true` | 输出模式（进度条）自动显示百分比 |
| 状态颜色 | 自动判断 | 根据值自动判断：>80% success, 50-80% warning, <50% danger |
| 进度条粗细 | `6px` | 标准粗细，视觉效果良好 |

## 三、使用场景分析

### 3.1 输入模式（编辑/新增表单）

**显示方式**：滑块（slider bar）

**默认行为**：
- 拖动时显示提示（带单位）
- 不显示输入框（保持简洁）
- 不显示刻度（保持简洁）

**使用示例**：
```go
// 进度字段：0-100，最简单用法
Progress int `json:"progress" gorm:"column:progress" widget:"name:完成进度;type:slider;min:0;max:100;unit:%" validate:"min=0,max=100"`

// 评分字段：0-10，步长0.5
Score float64 `json:"score" gorm:"column:score" widget:"name:评分;type:slider;min:0;max:10;step:0.5;unit:分" validate:"min=0,max=10"`

// 温度字段：-20到40，带单位
Temperature float64 `json:"temperature" gorm:"column:temperature" widget:"name:温度;type:slider;min:-20;max:40;unit:°C"`
```

### 3.2 输出模式（表格列表/详情）

**显示方式**：进度条（progress bar）

**默认行为**：
- 自动显示百分比
- 根据值自动判断状态颜色：
  - `>80%` → success（绿色）
  - `50-80%` → warning（黄色）
  - `<50%` → danger（红色）
- 标准进度条粗细（6px）

**使用示例**：
```go
// 进度字段：自动显示为进度条，带百分比和状态颜色
Progress int `json:"progress" gorm:"column:progress" widget:"name:完成进度;type:slider;min:0;max:100"`

// 评分字段：自动显示为进度条（相对于最大值10）
Score float64 `json:"score" gorm:"column:score" widget:"name:评分;type:slider;min:0;max:10"`
```

### 3.3 搜索模式

**显示方式**：范围输入框（两个输入框：最小值、最大值）

**搜索类型**：自动支持 `gte`（大于等于）和 `lte`（小于等于）

**默认行为**：
- 自动使用两个 `el-input-number` 实现范围搜索
- 最小值对应 `gte` 搜索
- 最大值对应 `lte` 搜索
- 自动限制在 min/max 范围内

**使用示例**：
```go
// 进度搜索：支持范围搜索
Progress int `json:"progress" gorm:"column:progress" widget:"name:完成进度;type:slider;min:0;max:100" search:"gte,lte"`
```

**URL 参数格式**：`gte=progress:50&lte=progress:80`（进度在50-80之间）

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
      :show-tooltip="true"
      :format-tooltip="formatTooltipFunc"
      :disabled="field.widget?.config?.disabled"
      @change="handleChange"
    />
    
    <!-- 输出模式：进度条 -->
    <el-progress
      v-else-if="mode === 'table-cell' || mode === 'detail'"
      :percentage="percentage"
      :status="autoStatus"
      :stroke-width="6"
      :show-text="true"
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

### 4.2 关键计算属性（自动处理）

```typescript
// 计算百分比（用于进度条显示）
const percentage = computed(() => {
  const value = props.value?.raw ?? 0
  const range = max - min
  if (range === 0) return 0
  return Math.round(((value - min) / range) * 100)
})

// 自动判断状态颜色（根据百分比）
const autoStatus = computed(() => {
  const pct = percentage.value
  if (pct > 80) return 'success'
  if (pct >= 50) return 'warning'
  return 'danger'
})

// 格式化提示（自动带上单位）
const formatTooltipFunc = computed(() => {
  const unit = config.value.unit || ''
  return (value: number) => {
    return unit ? `${value}${unit}` : String(value)
  }
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

Slider 组件设计遵循"极简原则"：
1. **参数最少**：只保留核心参数（min/max/step/default/unit）
2. **默认智能**：必要的功能默认开启，无需配置
3. **自动判断**：状态颜色、百分比显示等自动处理

**核心优势**：
- **学习成本低**：只需配置 min/max 即可使用
- **用户体验好**：滑块操作直观，进度条展示清晰
- **功能完整**：支持输入、输出、搜索三种模式
- **智能默认**：自动处理状态颜色、百分比等，无需手动配置

**使用示例（最简单）**：
```go
// 只需要 min 和 max，其他都自动处理
Progress int `json:"progress" widget:"name:完成进度;type:slider;min:0;max:100" search:"gte,lte"`
```

