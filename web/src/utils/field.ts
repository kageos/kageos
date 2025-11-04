/**
 * 字段工具函数
 */

import type { FieldConfig, FieldValue } from '@/core/types/field'
import { formatTimestamp } from './date'
import { widgetFactory } from '@/core/factories/WidgetFactory'

/**
 * 将原始值转换为 FieldValue 格式
 * 
 * 用于 TableRenderer 等场景，需要将后端返回的原始数据转换为统一的 FieldValue 格式
 * 
 * 🔥 优先使用 Widget 的 loadFromRawData 静态方法（如果存在）
 * 这样可以确保每个组件负责自己的数据转换逻辑
 * 
 * @param rawValue 原始值（来自后端）
 * @param field 字段配置
 * @returns FieldValue 格式的数据
 * 
 * @example
 * convertToFieldValue(1640995200000, { widget: { type: 'timestamp' } })
 * // { raw: 1640995200000, display: '2022-01-01 00:00:00', meta: {} }
 */
export function convertToFieldValue(rawValue: any, field: FieldConfig): FieldValue {
  // 如果已经是 FieldValue 格式，直接返回
  if (rawValue && typeof rawValue === 'object' && 'raw' in rawValue && 'display' in rawValue) {
    return rawValue as FieldValue
  }
  
  // ✅ 优先使用 Widget 的 loadFromRawData 静态方法（如果存在）
  const widgetType = field.widget?.type || 'input'
  const WidgetClass = widgetFactory.getWidgetClass(widgetType)
  
  // 检查 Widget 是否有 loadFromRawData 静态方法
  if (WidgetClass && typeof (WidgetClass as any).loadFromRawData === 'function') {
    try {
      return (WidgetClass as any).loadFromRawData(rawValue, field)
    } catch (error) {
      console.warn(`[convertToFieldValue] Widget.loadFromRawData failed for ${widgetType}:`, error)
      // 继续使用默认逻辑
    }
  }
  
  // 空值处理
  if (rawValue === null || rawValue === undefined) {
    return {
      raw: null,
      display: '-',
      meta: {}
    }
  }
  
  // 根据字段类型格式化 display
  let display = String(rawValue)
  
  // 时间戳类型：格式化日期
  if (field.widget?.type === 'timestamp') {
    display = formatTimestamp(rawValue, field.widget.config?.format)
  }
  
  // 数组类型：连接为字符串
  if (Array.isArray(rawValue)) {
    display = rawValue.join(', ')
  }
  
  return {
    raw: rawValue,
    display,
    meta: {}
  }
}
