/**
 * 验证器工具函数
 * 用于提取重复逻辑，提高代码复用性
 */

import type { FieldConfig, FieldValue } from '../../types/field'
import type { ValidationContext } from '../types'

/**
 * 判断字段值是否为空
 */
export function isEmpty(value: FieldValue): boolean {
  return value.raw === null ||
         value.raw === undefined ||
         value.raw === '' ||
         (Array.isArray(value.raw) && value.raw.length === 0)
}

/**
 * 从验证上下文中查找字段配置
 * 
 * @param context 验证上下文
 * @returns 字段配置，如果找不到则返回 null
 */
export function findFieldInContext(context: ValidationContext): FieldConfig | null {
  // 先尝试匹配 field_path，再尝试匹配 code
  let foundField = context.allFields.find(f => {
    if (f.field_path) {
      return f.field_path === context.fieldPath
    }
    return f.code === context.fieldPath
  })
  
  // 如果还找不到，尝试只匹配 code（可能 field_path 为空）
  if (!foundField) {
    foundField = context.allFields.find(f => f.code === context.fieldPath)
  }
  
  return foundField || null
}

/**
 * 生成必填错误消息
 * 
 * @param fieldName 字段名称（用户友好的名称）
 * @returns 错误消息
 */
export function createRequiredErrorMessage(fieldName: string): string {
  return `${fieldName}必填`
}

/**
 * 从验证上下文中获取字段名称
 * 
 * @param context 验证上下文
 * @param fallback 找不到字段时的默认名称
 * @returns 字段名称
 */
export function getFieldName(context: ValidationContext, fallback: string = '此字段'): string {
  const field = findFieldInContext(context)
  return field?.name || fallback
}

/**
 * 判断字段是否为字符串类型
 * 
 * 判断规则：
 * - 仅基于 data.type 判断（数据类型层面）
 * - data.type 包含 'string'、'text' 等认为是字符串类型
 * - 如果 data.type 为空，默认认为是字符串类型
 * 
 * 注意：不判断 widget.type，因为 widget.type 是渲染层面的概念，不是数据类型
 * 
 * @param field 字段配置（可为 null 或 undefined）
 * @returns 是否为字符串类型
 */
export function isStringField(field: FieldConfig | null | undefined): boolean {
  if (!field) return false
  
  const dataType = field.data?.type?.toLowerCase() || ''
  
  // 如果 data.type 为空，默认认为是字符串类型
  if (!dataType || dataType === '') {
    return true
  }
  
  // 检查 data.type 是否包含字符串类型标识
  if (dataType.includes('string') || dataType.includes('text')) {
    return true
  }
  
  return false
}

