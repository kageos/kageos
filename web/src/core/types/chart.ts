/**
 * Chart 相关类型定义
 * 与后端 types.Chart 对齐
 */

/**
 * Chart 数据结构（与后端 types.Chart 对齐）
 */
export interface Chart {
  // 图表类型（必需）
  chart_type: string  // 'bar' | 'line' | 'pie' | 'gauge' | 'scatter' | 'area'
  
  // 图表标题
  title?: string
  
  // X 轴数据（可选，某些图表类型不需要）
  x_axis?: string[]
  
  // 数据系列（必需）
  series: ChartSeries[]
  
  // ECharts 配置（可选，用于高级定制）
  echarts_config?: Record<string, any>
  
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
  // - bar/line/area: []interface{}，如 [100, 200, 150]
  // - pie: []map[string]interface{}，如 [{"name": "A", "value": 100}]
  // - gauge: []interface{}，如 [75]（单个数值，表示百分比）
  // - scatter: []interface{}，如 [[10, 20], [15, 25]]（二维数组，表示坐标点）
  data: any[]
  
  // 系列类型（可选，默认使用 ChartType）
  type?: string
  
  // 系列配置（可选，用于单个系列的样式配置）
  config?: Record<string, any>
}

