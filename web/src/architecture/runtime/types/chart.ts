/**
 * Chart 相关类型定义
 * 与后端图表接口返回结构对齐（后端使用 sdk/agent-app/chart 包 chart.LineChart/BarChart/PieChart/GaugeChart，序列化后结构一致）
 */

/**
 * Chart 数据结构（与后端图表接口返回的 JSON 对齐）
 */
export interface Chart {
  // 图表类型（必需），后端只支持 4 种：'line' | 'bar' | 'pie' | 'gauge'
  chart_type: string
  
  // 图表标题
  title?: string
  
  // X 轴数据（可选，某些图表类型不需要）
  x_axis?: string[]
  
  // 数据系列（必需）
  series: ChartSeries[]
  
  // 元数据（可选，用于扩展，使用中文键名）
  metadata?: Record<string, any>
  
  // 标识字段（用于类型识别）
  widget_type?: string  // 固定为 "chart"
  data_type?: string    // 固定为 "chart"
}

/**
 * ChartSeries 数据系列
 */
export interface ChartSeries {
  // 系列名称
  name: string
  
  // 数据点（必需）
  // 不同类型图表的数据格式：
  // - line/bar: 与 x_axis 一一对应，如 [100, 200, 150]
  // - pie: []{ name, value }，如 [{"name": "A", "value": 100}]
  // - gauge: 单值，如 [75]
  data: any[]
  
  // 系列类型（可选，后端会在返回前注入，与 chart_type 一致）
  type?: string
  
  // 系列配置（可选，用于单个系列的样式配置）
  config?: Record<string, any>
}

