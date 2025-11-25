/**
 * 字段工具函数
 */

import type { FieldConfig, FieldValue } from '@/core/types/field'
import { formatTimestamp } from './date'
import { WidgetType } from '@/core/constants/widget'

/**
 * 将原始值转换为 FieldValue 格式
 * 
 * 用于 TableRenderer 等场景，需要将后端返回的原始数据转换为统一的 FieldValue 格式
 * 
 * 🔥 重构说明：
 * - 移除了对旧版本 widgetFactory 的依赖
 * - 直接根据 widget.type 进行转换，不依赖 Widget 类
 * - 保持与 v2 组件兼容的数据格式
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
  
  // 空值处理
  if (rawValue === null || rawValue === undefined) {
    return {
      raw: null,
      display: '-',
      meta: {}
    }
  }
  
  const widgetType = field.widget?.type || WidgetType.INPUT
  
  // 根据 widget 类型进行转换
  let display = String(rawValue)
  
  // 时间戳类型：格式化日期
  if (widgetType === WidgetType.TIMESTAMP) {
    display = formatTimestamp(rawValue, field.widget.config?.format)
  }
  
  // 数组类型：连接为字符串
  else if (Array.isArray(rawValue)) {
    display = rawValue.join(', ')
  }
  
  // 布尔类型：转换为中文显示
  else if (typeof rawValue === 'boolean') {
    display = rawValue ? '是' : '否'
  }
  
  // 数字类型：保持原样（v2 组件会自己格式化）
  else if (typeof rawValue === 'number') {
    display = String(rawValue)
  }
  
  // 对象类型：转换为 JSON 字符串（用于调试）
  else if (typeof rawValue === 'object') {
    try {
      display = JSON.stringify(rawValue)
    } catch {
      display = String(rawValue)
    }
  }
  
  return {
    raw: rawValue,
    display,
    meta: {}
  }
}
