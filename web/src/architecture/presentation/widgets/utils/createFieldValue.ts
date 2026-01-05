/**
 * 创建 FieldValue 的工具函数
 * 🔥 统一处理所有组件的 FieldValue 创建，确保所有组件都设置 dataType 和 widgetType
 * 
 * 设计原则：
 * - dataType 和 widgetType 是通用字段，所有组件都应该设置
 * - 方便后续在提交前做类型判断和转换
 * - 避免特殊逻辑，支持未来更复杂的场景
 */

import type { FieldConfig, FieldValue } from '@/architecture/domain/types'

/**
 * 创建 FieldValue
 * 
 * @param field 字段配置
 * @param raw 原始值
 * @param display 显示值
 * @param meta 元数据（可选）
 * @returns FieldValue
 */
export function createFieldValue(
  field: FieldConfig,
  raw: any,
  display: string,
  meta?: Record<string, any>
): FieldValue {
  return {
    raw,
    display,
    dataType: field.data?.type,  // 🔥 数据类型（通用字段，和 display 同级别）
    widgetType: field.widget?.type,  // 🔥 组件类型（通用字段，和 display 同级别）
    meta: meta || {}
  }
}

/**
 * 创建默认的 FieldValue（空值）
 */
export function createEmptyFieldValue(field: FieldConfig): FieldValue {
  return createFieldValue(field, null, '', {})
}

